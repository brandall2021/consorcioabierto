package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brandall2021/consorcioabierto/internal/cobranzas"
)

func TestWriteCobranzaErrorMapsReversalConflict(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/cobranzas/123/revertir", nil)

	(&AuthHandlers{}).writeCobranzaError(rec, req, cobranzas.ErrCobranzaNoRevertible)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status unexpected: %d", rec.Code)
	}
	var problem struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "conflict" {
		t.Fatalf("problem code unexpected: %s", problem.Code)
	}
	if problem.Detail == "" {
		t.Fatal("expected detail text")
	}
	if !errors.Is(cobranzas.ErrCobranzaNoRevertible, cobranzas.ErrCobranzaNoRevertible) {
		t.Fatal("sanity check failed")
	}
}

func TestWriteReceiptPDFResponseSetsHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	writeReceiptPDFResponse(rec, "recibo-123.pdf", []byte("%PDF-1.4 test"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status unexpected: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("content-type unexpected: %s", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != "attachment; filename=\"recibo-123.pdf\"" {
		t.Fatalf("content-disposition unexpected: %s", got)
	}
	if body := rec.Body.String(); body != "%PDF-1.4 test" {
		t.Fatalf("body unexpected: %s", body)
	}
}
