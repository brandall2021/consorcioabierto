package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/cobranzas"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/identity"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func mustClaims(w http.ResponseWriter, r *http.Request) *identity.Claims {
	c := claimsFrom(r.Context())
	if c == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return nil
	}
	return c
}

func (h *AuthHandlers) ListCobranzas(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := cobranzas.ListCobranzas(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"data": items, "meta": map[string]any{"request_id": middleware.GetReqID(r.Context())}})
}

func (h *AuthHandlers) CreateCobranza(w http.ResponseWriter, r *http.Request) {
	var in cobranzas.CobranzaInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "crear cobranzas es idempotente", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()
	claims := mustClaims(w, r)
	if claims == nil {
		return
	}

	item, err := cobranzas.CreateCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "cobranzas.create", RecursoType: "payment", RecursoID: item.ID, Diff: map[string]any{"unidad_id": item.UnidadID, "importe_cents": item.Importe.AmountCents, "canal": item.Canal}}))
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) CreateMercadoPagoCobranza(w http.ResponseWriter, r *http.Request) {
	var in cobranzas.CobranzaInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "crear cobranzas es idempotente", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()
	claims := mustClaims(w, r)
	if claims == nil {
		return
	}

	item, err := cobranzas.CreateMercadoPagoCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in, idemKey, h.PSP)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "cobranzas.create_mercado_pago", RecursoType: "payment", RecursoID: item.Cobranza.ID, Diff: map[string]any{"unidad_id": item.Cobranza.UnidadID, "importe_cents": item.Cobranza.Importe.AmountCents, "checkout_url": item.CheckoutURL}}))
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) GetCobranza(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	detalle, err := cobranzas.GetCobranzaDetalle(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, detalle)
}

func (h *AuthHandlers) GetCobranzaRecibo(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	detalle, err := cobranzas.GetCobranzaDetalle(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}

	pdf := cobranzas.BuildReciboPDF(detalle)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=recibo.pdf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func (h *AuthHandlers) GetCuentaCorriente(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	resp, err := cobranzas.GetCuentaCorriente(r.Context(), q, chi.URLParam(r, "id"), chi.URLParam(r, "unidadId"))
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	resp.Meta.RequestID = middleware.GetReqID(r.Context())
	httpapi.WriteJSON(w, http.StatusOK, resp)
}

func (h *AuthHandlers) AcreditarCobranza(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Asignaciones []cobranzas.AsignacionInput `json:"asignaciones"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "acreditar cobranzas es idempotente", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()
	claims := mustClaims(w, r)
	if claims == nil {
		return
	}

	detalle, err := cobranzas.AcreditarCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in.Asignaciones, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "cobranzas.acreditar", RecursoType: "payment", RecursoID: detalle.Cobranza.ID, Diff: map[string]any{"saldo_a_favor_cents": detalle.SaldoAFavorCents}}))
	httpapi.WriteJSON(w, http.StatusOK, detalle)
}

func (h *AuthHandlers) AjustarAsignaciones(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Asignaciones []cobranzas.AsignacionInput `json:"asignaciones"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "ajustar asignaciones es idempotente", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()
	claims := mustClaims(w, r)
	if claims == nil {
		return
	}

	detalle, err := cobranzas.AjustarAsignaciones(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in.Asignaciones, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "cobranzas.asignaciones", RecursoType: "payment", RecursoID: detalle.Cobranza.ID, Diff: map[string]any{"saldo_a_favor_cents": detalle.SaldoAFavorCents}}))
	httpapi.WriteJSON(w, http.StatusOK, detalle)
}

func (h *AuthHandlers) RevertirCobranza(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "revertir cobranzas es idempotente", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()
	claims := mustClaims(w, r)
	if claims == nil {
		return
	}

	detalle, err := cobranzas.RevertirCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "cobranzas.revertir", RecursoType: "payment", RecursoID: detalle.Cobranza.ID}))
	httpapi.WriteJSON(w, http.StatusOK, detalle)
}

func (h *AuthHandlers) writeCobranzaError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, cobranzas.ErrCobranzaInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, cobranzas.ErrCobranzaNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Cobranza no encontrada", err.Error(), nil)
	case errors.Is(err, cobranzas.ErrCobranzaDuplicateReferencia), errors.Is(err, cobranzas.ErrIdempotencyConflict):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	default:
		slog.Error("cobranzas", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}
