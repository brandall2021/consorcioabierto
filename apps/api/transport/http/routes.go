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
	r.Post("/auth/mfa/setup", h.MfaSetup)
	r.Post("/auth/mfa/confirm", h.MfaConfirm)
	r.Post("/auth/mfa/verify", h.MfaVerify)
	r.Post("/auth/mfa/disable", h.MfaDisable)

	r.Group(func(authed chi.Router) {
		authed.Use(RequireAuth(h.Manager))
		authed.Get("/me", h.Me)
		authed.Get("/memberships", h.Memberships)
	})

	r.Route("/audit-events", func(ar chi.Router) {
		ar.Use(RequirePermission(h.Manager, "auditoria.read"))
		ar.Get("/", h.listAuditEventsHandler())
	})

	// Consorcios: lectura con consorcios.read, escritura con consorcios.manage.
	r.Group(func(cr chi.Router) {
		cr.Use(RequirePermission(h.Manager, "consorcios.read"))
		cr.Get("/consorcios", h.ListConsorcios)
		cr.Get("/consorcios/{id}", h.GetConsorcio)
		cr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "consorcios.manage"))
			mgmt.Post("/consorcios", h.CreateConsorcio)
			mgmt.Patch("/consorcios/{id}", h.UpdateConsorcio)
		})
	})
}
