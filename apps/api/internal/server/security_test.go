package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"multipurpose-booking-system/api/internal/service"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSecurityLimiterConcurrentAndCanonicalPeers(t *testing.T) {
	limiter := newAttemptLimiter(10)
	var accepted atomic.Int32
	var workers sync.WaitGroup
	for i := 0; i < 100; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			peer := "127.0.0.1:1000"
			if i%2 == 0 {
				peer = "[::ffff:127.0.0.1]:2000"
			}
			if limiter.allow(peer) {
				accepted.Add(1)
			}
		}(i)
	}
	workers.Wait()
	if accepted.Load() != 10 {
		t.Fatalf("accepted %d attempts", accepted.Load())
	}
	now := time.Now()
	limiter.now = func() time.Time { return now.Add(16 * time.Minute) }
	if !limiter.allow("127.0.0.1:3000") {
		t.Fatal("expired window not reset")
	}
	limiter = newAttemptLimiter(1)
	for i := 0; i < 10000; i++ {
		limiter.windows[string(rune(i))] = registrationWindow{count: 1, until: now.Add(time.Hour)}
	}
	if limiter.allow("192.0.2.1:80") || len(limiter.windows) != 10000 {
		t.Fatal("capacity not bounded")
	}
	limiter.now = func() time.Time { return now.Add(2 * time.Hour) }
	if !limiter.allow("192.0.2.1:80") || len(limiter.windows) != 1 {
		t.Fatal("expired entries not pruned")
	}
}
func TestSecurityLoginEnumerationAndProxySpoofing(t *testing.T) {
	var baseline string
	for _, failure := range []error{service.ErrInvalidCredentials, service.ErrAccountDisabled, service.ErrInvalidSession} {
		h, err := NewLoginHandler(&loginStub{err: failure}, "http://localhost:3000", slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
		if err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, loginRequest(loginBody, "http://localhost:3000"))
		var body struct {
			Error map[string]string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		delete(body.Error, "request_id")
		data, _ := json.Marshal(body)
		if rec.Code != 401 || len(rec.Result().Cookies()) != 0 {
			t.Fatal("credential failure exposed state")
		}
		if baseline == "" {
			baseline = string(data)
		} else if baseline != string(data) {
			t.Fatal("enumerable failure response")
		}
	}
	stub := &loginStub{err: service.ErrInvalidCredentials}
	h, _ := NewLoginHandler(stub, "http://localhost:3000", slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	for i := 0; i < 11; i++ {
		req := loginRequest(loginBody, "http://localhost:3000")
		req.Header.Set("X-Forwarded-For", time.Unix(int64(i), 0).String())
		req.Header.Set("X-Real-IP", "192.0.2.2")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if i == 10 && rec.Code != 429 {
			t.Fatal("forwarding headers bypass limit")
		}
	}
	if stub.calls != 10 {
		t.Fatal("limited request reached credentials")
	}
}
