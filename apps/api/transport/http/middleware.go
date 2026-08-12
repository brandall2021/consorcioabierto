package http

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/identity"
)

type claimsContextKey struct{}

// withClaims guarda las claims verificadas en el contexto de la request.
func withClaims(ctx context.Context, c *identity.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, c)
}

// claimsFrom devuelve las claims puestas por RequireAuth, o nil.
func claimsFrom(ctx context.Context) *identity.Claims {
	c, _ := ctx.Value(claimsContextKey{}).(*identity.Claims)
	return c
}

// Authorizer verifica access tokens y resuelve permisos. AuthManager lo
// implementa; la interfaz permite testear el middleware sin base de datos.
type Authorizer interface {
	VerifyAccessToken(token string) (*identity.Claims, error)
	PermissionsForClaims(ctx context.Context, claims *identity.Claims) ([]string, error)
}

// bearerToken extrae el access token del header Authorization o de la cookie.
func bearerToken(r *http.Request) string {
	if tok := r.Header.Get("Authorization"); tok != "" {
		return tok
	}
	if c, err := r.Cookie(accessCookieName); err == nil {
		return c.Value
	}
	return ""
}

// RequireAuth es middleware de autenticación: verifica el access token
// (Bearer o cookie) y deja las claims en el contexto. No autoriza.
func RequireAuth(a Authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := a.VerifyAccessToken(bearerToken(r))
			if err != nil {
				httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
				return
			}
			next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
		})
	}
}

// RequirePermission es middleware de autorización: exige un permiso efectivo
// para la membresía activa del token ([ADR-0009]). El permiso se resuelve con
// caché de membresías en el backend, nunca se confía en claims del cliente.
func RequirePermission(a Authorizer, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireAuth(a)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := claimsFrom(r.Context())
			if claims == nil {
				httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
				return
			}
			perms, err := a.PermissionsForClaims(r.Context(), claims)
			if err != nil {
				if errors.Is(err, identity.ErrMembershipNotFound) || errors.Is(err, identity.ErrMembershipInactive) {
					httpapi.WriteProblem(w, r, http.StatusForbidden, "membership_inactive", "Membresía inválida", err.Error(), nil)
					return
				}
				httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
				return
			}
			if !slices.Contains(perms, permission) {
				httpapi.WriteProblem(w, r, http.StatusForbidden, "forbidden", "Sin permisos", "Se requiere permiso: "+permission, nil)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}
