package http

import (
	"github.com/go-chi/chi/v5"
)

// RegisterAuthRoutes registra el sub-router de identidad del contrato OpenAPI.
func RegisterAuthRoutes(r chi.Router, h *AuthHandlers) {
	r.Post("/webhooks/mercado-pago", h.MercadoPagoWebhook)
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
		authed.Get("/portal", h.GetPortalHome)
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

	// Cobranzas: lectura con cobranzas.read, alta manual con cobranzas.manage.
	r.Group(func(cr chi.Router) {
		cr.Use(RequirePermission(h.Manager, "cobranzas.read"))
		cr.Get("/consorcios/{id}/cobranzas", h.ListCobranzas)
		cr.Get("/consorcios/{id}/morosidad", h.GetMorosidad)
		cr.Get("/consorcios/{id}/unidades/{unidadId}/cuenta-corriente", h.GetCuentaCorriente)
		cr.Get("/cobranzas/{id}", h.GetCobranza)
		cr.Get("/cobranzas/{id}/recibo.pdf", h.GetCobranzaRecibo)
		cr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "cobranzas.manage"))
			mgmt.Post("/consorcios/{id}/cobranzas", h.CreateCobranza)
			mgmt.Post("/consorcios/{id}/cobranzas/mercado-pago", h.CreateCobranzaMercadoPago)
			mgmt.Post("/cobranzas/{id}/acreditar", h.AcreditarCobranza)
			mgmt.Post("/cobranzas/{id}/asignaciones", h.AjustarAsignaciones)
			mgmt.Post("/cobranzas/{id}/revertir", h.RevertirCobranza)
		})
	})

	// Comunicaciones: lectura y publicación con comunicaciones.*.
	r.Group(func(cmr chi.Router) {
		cmr.Use(RequirePermission(h.Manager, "comunicaciones.read"))
		cmr.Get("/consorcios/{id}/comunicados", h.ListComunicados)
		cmr.Group(func(send chi.Router) {
			send.Use(RequirePermission(h.Manager, "comunicaciones.send"))
			send.Post("/consorcios/{id}/comunicados", h.CreateComunicado)
			send.Post("/comunicados/{id}/publicar", h.PublicarComunicado)
		})
	})

	// Reclamos: lectura general, creación/mensajes para consorcista o manage, transiciones para manage.
	r.Group(func(rr chi.Router) {
		rr.Use(RequirePermission(h.Manager, "reclamos.read"))
		rr.Get("/consorcios/{id}/reclamos", h.ListReclamos)
		rr.Get("/reclamos/{id}", h.GetReclamo)
		rr.Post("/consorcios/{id}/reclamos", h.CreateReclamo)
		rr.Post("/reclamos/{id}/mensajes", h.AddReclamoMensaje)
		rr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "reclamos.manage"))
			mgmt.Post("/reclamos/{id}/transiciones", h.TransicionarReclamo)
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

	// Expensas: conceptos con expensas.read/create, gastos con gastos.read/manage.
	r.Group(func(er chi.Router) {
		er.Use(RequirePermission(h.Manager, "expensas.read"))
		er.Get("/consorcios/{id}/conceptos", h.ListConceptos)
		er.Get("/conceptos/{id}", h.GetConcepto)
		er.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "expensas.create"))
			mgmt.Post("/consorcios/{id}/conceptos", h.CreateConcepto)
		})
	})

	r.Group(func(gr chi.Router) {
		gr.Use(RequirePermission(h.Manager, "gastos.read"))
		gr.Get("/consorcios/{id}/gastos", h.ListGastos)
		gr.Get("/consorcios/{consorcioId}/gastos/{gastoId}", h.GetGasto)
		gr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "gastos.manage"))
			mgmt.Post("/consorcios/{id}/gastos", h.CreateGasto)
		})
	})

	// Liquidaciones: lectura con expensas.read, creación/cálculo con expensas.create,
	// confirmar/anular con expensas.confirm y publicar con expensas.publish.
	r.Group(func(lr chi.Router) {
		lr.Use(RequirePermission(h.Manager, "expensas.read"))
		lr.Get("/consorcios/{id}/liquidaciones", h.ListLiquidaciones)
		lr.Get("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}", h.GetLiquidacion)
		lr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "expensas.create"))
			mgmt.Post("/consorcios/{id}/liquidaciones", h.CreateLiquidacion)
			mgmt.Patch("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}", h.PatchLiquidacion)
			mgmt.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/calcular", h.CalcularLiquidacionHandler)
		})
		lr.Group(func(conf chi.Router) {
			conf.Use(RequirePermission(h.Manager, "expensas.confirm"))
			conf.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/confirmar", h.ConfirmarLiquidacionHandler)
			conf.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/anular", h.AnularLiquidacionHandler)
		})
		lr.Group(func(pub chi.Router) {
			pub.Use(RequirePermission(h.Manager, "expensas.publish"))
			pub.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/publicar", h.PublicarLiquidacionHandler)
		})
	})
}
