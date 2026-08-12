package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer() http.Handler {
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), "test", nil)
}

func TestHealthzRoot(t *testing.T) {
	h := newTestServer()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("body = %q, se esperaba ok", body)
	}
}

func TestHealthAPIV1(t *testing.T) {
	h := newTestServer()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status = %q, se esperaba ok", payload["status"])
	}
}

func TestHealthAPIV1MandaRequestID(t *testing.T) {
	h := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Request-ID", "550e8400-e29b-41d4-a716-446655440000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rec.Code)
	}
}
