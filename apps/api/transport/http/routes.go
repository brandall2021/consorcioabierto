package http

import (
	"github.com/go-chi/chi/v5"
)

// RegisterAuthRoutes registra el sub-router de identidad del contrato OpenAPI.
func RegisterAuthRoutes(r chi.Router, h *AuthHandlers) {
	r.Post("/auth/login", h.Login)
	r.Post("/auth/select-tenant", h.SelectTenant)
	r.Post("/auth/refresh", h.Refresh)
	r.Post("/auth/logout", h.Logout)
	r.Get("/me", h.Me)
	r.Get("/memberships", h.Memberships)
}
