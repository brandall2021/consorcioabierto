package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// bearerOrCookie debe devolver el JWT crudo: si deja el prefijo "Bearer " puesto,
// la verificación de firma falla con "firma inválida" en las rutas que la usan
// (MFA setup/confirm/disable y selección de tenant).
func TestBearerOrCookieQuitaElPrefijoBearer(t *testing.T) {
	h := &AuthHandlers{}
	casos := []struct {
		header string
		want   string
		nombre string
	}{
		{"Bearer abc.def.ghi", "abc.def.ghi", "con prefijo Bearer"},
		{"bearer abc.def.ghi", "abc.def.ghi", "prefijo en minúsculas"},
		{"BEARER abc.def.ghi", "abc.def.ghi", "prefijo en mayúsculas"},
		{"  Bearer   abc.def.ghi", "abc.def.ghi", "espacios alrededor"},
		{"abc.def.ghi", "abc.def.ghi", "token crudo sin prefijo"},
		{"Bearer   ", "", "solo prefijo sin token"},
	}
	for _, c := range casos {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", c.header)
		if got := h.bearerOrCookie(req); got != c.want {
			t.Errorf("%s: bearerOrCookie(%q) = %q, se esperaba %q", c.nombre, c.header, got, c.want)
		}
	}
}

func TestBearerOrCookieUsaLaCookie(t *testing.T) {
	h := &AuthHandlers{}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: accessCookieName, Value: "abc.def.ghi"})
	if got := h.bearerOrCookie(req); got != "abc.def.ghi" {
		t.Fatalf("bearerOrCookie con cookie = %q, se esperaba %q", got, "abc.def.ghi")
	}
}

func TestBearerOrCookieSinToken(t *testing.T) {
	h := &AuthHandlers{}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if got := h.bearerOrCookie(req); got != "" {
		t.Fatalf("bearerOrCookie sin token = %q, se esperaba %q", got, "")
	}
}