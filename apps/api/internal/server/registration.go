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
	"net/http"
	"time"

	"multipurpose-booking-system/api/internal/service"
)

type registrar interface {
	Register(context.Context, service.RegistrationInput) error
}
type RegistrationHandler struct {
	attemptLimiter
	registration registrar
	origin       string
	logger       *slog.Logger
	slots        chan struct{}
}

func NewRegistrationHandler(registration registrar, origin string, logger *slog.Logger) *RegistrationHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &RegistrationHandler{attemptLimiter: newAttemptLimiter(5), registration: registration, origin: origin, logger: logger, slots: make(chan struct{}, 2)}
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
