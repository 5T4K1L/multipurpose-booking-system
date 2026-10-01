package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"multipurpose-booking-system/api/internal/service"
)

type registrar interface {
	Register(context.Context, service.RegistrationInput) error
}
type registrationWindow struct {
	count int
	until time.Time
}
type RegistrationHandler struct {
	registration registrar
	origin       string
	logger       *slog.Logger
	mu           sync.Mutex
	windows      map[string]registrationWindow
	slots        chan struct{}
	now          func() time.Time
}

func NewRegistrationHandler(registration registrar, origin string, logger *slog.Logger) *RegistrationHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &RegistrationHandler{registration: registration, origin: origin, logger: logger, windows: make(map[string]registrationWindow), slots: make(chan struct{}, 2), now: time.Now}
}

// Only the transport peer is trusted. Forwarded headers cannot bypass this limiter.
func (h *RegistrationHandler) allow(peer string) bool {
	host, _, err := net.SplitHostPort(peer)
	if err != nil {
		host = peer
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		host = "unknown"
	} else {
		host = address.Unmap().String()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	window, exists := h.windows[host]
	if !exists && len(h.windows) >= 10000 {
		for key, value := range h.windows {
			if !now.Before(value.until) {
				delete(h.windows, key)
			}
		}
		if len(h.windows) >= 10000 {
			return false
		}
	}
	if !now.Before(window.until) {
		window = registrationWindow{until: now.Add(15 * time.Minute)}
	}
	if window.count >= 5 {
		return false
	}
	window.count++
	h.windows[host] = window
	return true
}

func (h *RegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Origin")
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	requestID := hex.EncodeToString(randomID[:])
	w.Header().Set("X-Request-ID", requestID)
	fail := func(status int, code, message string) {
		h.logger.Info("registration", "request_id", requestID, "outcome", code)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": requestID}})
	}
	origin := r.Header.Get("Origin")
	if origin != "" {
		if origin != h.origin {
			fail(http.StatusForbidden, "ORIGIN_REJECTED", "Request origin is not allowed.")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, OPTIONS")
		fail(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Use POST.")
		return
	}
	if !h.allow(r.RemoteAddr) {
		w.Header().Set("Retry-After", "900")
		fail(http.StatusTooManyRequests, "RATE_LIMITED", "Please try again later.")
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		fail(http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Use application/json.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input service.RegistrationInput
	err = decoder.Decode(&input)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = errors.New("unexpected trailing content")
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body is too large.")
			return
		}
		fail(http.StatusBadRequest, "INVALID_JSON", "Provide one valid JSON registration object.")
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		w.Header().Set("Retry-After", "1")
		fail(http.StatusTooManyRequests, "RATE_LIMITED", "Please try again later.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	err = h.registration.Register(ctx, input)
	if errors.Is(err, service.ErrInvalidRegistration) {
		fail(http.StatusBadRequest, "VALIDATION_ERROR", service.ErrInvalidRegistration.Error())
		return
	}
	if err != nil {
		fail(http.StatusServiceUnavailable, "REGISTRATION_UNAVAILABLE", "Registration is temporarily unavailable.")
		return
	}
	// Creation and duplicates have identical public responses and log outcomes.
	h.logger.Info("registration", "request_id", requestID, "outcome", "accepted")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Registration request received."})
}
