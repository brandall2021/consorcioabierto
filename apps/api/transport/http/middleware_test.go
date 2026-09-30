package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brandall2021/consorcioabierto/internal/identity"
)

type stubAuthorizer struct {
	claims   *identity.Claims
	perms    []string
	err      error
	permErr  error
	received string
}

func (s *stubAuthorizer) VerifyAccessToken(token string) (*identity.Claims, error) {
	s.received = token
	if s.err != nil {
		return nil, s.err
	}
	if token == "" {
		return nil, errors.New("token ausente")
	}
	return s.claims, nil
}

func (s *stubAuthorizer) PermissionsForClaims(ctx context.Context, claims *identity.Claims) ([]string, error) {
	if s.permErr != nil {
		return nil, s.permErr
	}
	return s.perms, nil
}

func callMiddleware(next http.Handler, auth Authorizer, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	next.ServeHTTP(rec, req)
	return rec
}

func okHandler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claimsFrom(r.Context()) == nil {
			t.Error("handler sin claims en contexto")
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestRequireAuthSinToken(t *testing.T) {
	next := RequireAuth(&stubAuthorizer{})(okHandler(t))
	rec := callMiddleware(next, nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", rec.Code)
	}
}

func TestBearerTokenQuitaElPrefijoBearer(t *testing.T) {
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
		if got := bearerToken(req); got != c.want {
			t.Errorf("%s: bearerToken(%q) = %q, se esperaba %q", c.nombre, c.header, got, c.want)
		}
	}
}

// El token del header Authorization debe llegar al Authorizer sin el prefijo
// "Bearer ": si no, la verificación de firma falla con "firma inválida".
func TestRequireAuthPasaElTokenSinPrefijoAlAuthorizer(t *testing.T) {
	auth := &stubAuthorizer{claims: &identity.Claims{Subject: "u1"}}
	next := RequireAuth(auth)(okHandler(t))
	rec := callMiddleware(next, auth, map[string]string{"Authorization": "Bearer abc.def.ghi"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, se esperaba 204", rec.Code)
	}
	if auth.received != "abc.def.ghi" {
		t.Fatalf("el Authorizer recibió %q, se esperaba %q", auth.received, "abc.def.ghi")
	}
}

func TestRequireAuthTokenInvalido(t *testing.T) {
	auth := &stubAuthorizer{err: errors.New("firma inválida")}
	next := RequireAuth(auth)(okHandler(t))
	rec := callMiddleware(next, auth, map[string]string{"Authorization": "aaa.bbb.ccc"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", rec.Code)
	}
}

func TestRequireAuthConCookie(t *testing.T) {
	auth := &stubAuthorizer{claims: &identity.Claims{Subject: "u1"}}
	next := RequireAuth(auth)(okHandler(t))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: accessCookieName, Value: "token"})
	rec := httptest.NewRecorder()
	next.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, se esperaba 204", rec.Code)
	}
}

func TestRequirePermissionConPermiso(t *testing.T) {
	auth := &stubAuthorizer{
		claims: &identity.Claims{Subject: "u1", Membership: "m1"},
		perms:  []string{"consorcios.read"},
	}
	next := RequirePermission(auth, "consorcios.read")(okHandler(t))
	rec := callMiddleware(next, auth, map[string]string{"Authorization": "token"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, se esperaba 204", rec.Code)
	}
}

func TestRequirePermissionSinPermiso(t *testing.T) {
	auth := &stubAuthorizer{
		claims: &identity.Claims{Subject: "u1", Membership: "m1"},
		perms:  []string{"consorcios.read"},
	}
	next := RequirePermission(auth, "expensas.manage")(okHandler(t))
	rec := callMiddleware(next, auth, map[string]string{"Authorization": "token"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, se esperaba 403", rec.Code)
	}
}

func TestRequirePermissionMembresiaInactiva(t *testing.T) {
	auth := &stubAuthorizer{
		claims:  &identity.Claims{Subject: "u1", Membership: "m1"},
		permErr: identity.ErrMembershipInactive,
	}
	next := RequirePermission(auth, "consorcios.read")(okHandler(t))
	rec := callMiddleware(next, auth, map[string]string{"Authorization": "token"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, se esperaba 403", rec.Code)
	}
}
