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

	// Unidades: lectura con ufs.read, escritura con ufs.manage.
	r.Group(func(ur chi.Router) {
		ur.Use(RequirePermission(h.Manager, "ufs.read"))
		ur.Get("/consorcios/{id}/unidades", h.ListUnidades)
		ur.Get("/import-jobs/{id}", h.GetImportJob)
		ur.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "ufs.manage"))
			mgmt.Post("/consorcios/{id}/unidades", h.CreateUnidad)
			mgmt.Post("/consorcios/{id}/unidades/import-jobs", h.CreateImportJob)
			mgmt.Post("/import-jobs/{id}/confirm", h.ConfirmImportJob)
		})
	})

	// Proveedores: lectura con proveedores.read, escritura con proveedores.manage.
	r.Group(func(pr chi.Router) {
		pr.Use(RequirePermission(h.Manager, "proveedores.read"))
		pr.Get("/consorcios/{id}/proveedores", h.ListProveedores)
		pr.Get("/proveedores/{id}", h.GetProveedor)
		pr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "proveedores.manage"))
			mgmt.Post("/consorcios/{id}/proveedores", h.CreateProveedor)
			mgmt.Patch("/proveedores/{id}", h.UpdateProveedor)
			mgmt.Patch("/proveedores/{id}/estado", h.SetProveedorEstado)
		})
	})

	// Documentos: lectura con documentos.read, subida con documentos.manage.
	r.Group(func(dr chi.Router) {
		dr.Use(RequirePermission(h.Manager, "documentos.read"))
		dr.Get("/documentos/{id}/download-url", h.GetDocumentDownloadUrl)
		dr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "documentos.manage"))
			mgmt.Post("/document-upload-intents", h.CreateDocumentUploadIntent)
		})
	})
}
