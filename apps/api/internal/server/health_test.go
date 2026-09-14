package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthHandler verifies a successful response from the HealthHandler.
func TestHealthHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	HealthHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, response.Code)
	}

	if response.Body.String() != `{"status":"ok"}`+"\n" {
		t.Fatalf("expected body %q, got %q", `{"status":"ok"}`, response.Body.String())
	}
}

// TestHealthHandlerRejectsUnsupportedMethods verifies non-GET requests fail.
func TestHealthHandlerRejectsUnsupportedMethods(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/health", nil)
	response := httptest.NewRecorder()

	HealthHandler(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status code %d, got %d", http.StatusMethodNotAllowed, response.Code)
	}
}
