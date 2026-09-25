package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/cobranzas"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type asignacionesRequest struct {
	Asignaciones []cobranzas.AllocationInput `json:"asignaciones"`
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
		h.writeCobranzaError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

func (h *AuthHandlers) GetCobranza(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	item, err := cobranzas.GetCobranzaDetalle(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) GetCobranzaRecibo(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	item, err := cobranzas.GetCobranzaDetalle(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	pdf, err := cobranzas.GenerateReciboPDF(item.Cobranza, item.Asignaciones)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	writeReceiptPDFResponse(w, fmt.Sprintf("recibo-%s.pdf", item.Cobranza.ID), pdf)
}

func (h *AuthHandlers) CreateCobranza(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "El alta de cobranzas es idempotente (ADR-0004)", nil)
		return
	}

	var in cobranzas.CobranzaInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := cobranzas.CreateCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "cobranzas.create",
		RecursoType: "cobranza",
		RecursoID:   item.ID,
		Diff: map[string]any{
			"unidad_id":  item.UnidadID,
			"importe":    item.Importe.AmountCents,
			"referencia": item.Referencia,
		},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) CreateCobranzaMercadoPago(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "El alta de cobranzas es idempotente (ADR-0004)", nil)
		return
	}

	var in cobranzas.CobranzaInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := cobranzas.CreateMercadoPagoCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in, idemKey, h.PSP)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "cobranzas.mercadopago.create",
		RecursoType: "cobranza",
		RecursoID:   item.Cobranza.ID,
		Diff: map[string]any{
			"unidad_id":     item.Cobranza.UnidadID,
			"importe":       item.Cobranza.Importe.AmountCents,
			"checkout_url":  item.CheckoutURL,
			"provider":      item.Provider,
			"preference_id": item.PreferenceID,
		},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) AcreditarCobranza(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "La acreditación es idempotente (ADR-0004)", nil)
		return
	}

	var in asignacionesRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := cobranzas.AcreditarCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in.Asignaciones, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "cobranzas.acreditar",
		RecursoType: "cobranza",
		RecursoID:   item.Cobranza.ID,
		Diff: map[string]any{
			"saldo_a_favor_cents": item.SaldoAFavorCents,
			"asignaciones":        len(item.Asignaciones),
		},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) AjustarAsignaciones(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "El ajuste es idempotente (ADR-0004)", nil)
		return
	}

	var in asignacionesRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := cobranzas.AjustarAsignaciones(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, in.Asignaciones, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "cobranzas.asignaciones",
		RecursoType: "cobranza",
		RecursoID:   item.Cobranza.ID,
		Diff: map[string]any{
			"saldo_a_favor_cents": item.SaldoAFavorCents,
			"asignaciones":        len(item.Asignaciones),
		},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) RevertirCobranza(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "La reversa es idempotente (ADR-0004)", nil)
		return
	}

	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := cobranzas.RevertirCobranza(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, idemKey)
	if err != nil {
		h.writeCobranzaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "cobranzas.revertir",
		RecursoType: "cobranza",
		RecursoID:   item.Cobranza.ID,
		Diff: map[string]any{
			"saldo_a_favor_cents": item.SaldoAFavorCents,
			"asignaciones":        len(item.Asignaciones),
		},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) writeCobranzaError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, cobranzas.ErrCobranzaInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, cobranzas.ErrCobranzaNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Cobranza no encontrada", err.Error(), nil)
	case errors.Is(err, cobranzas.ErrCobranzaEstadoInvalido), errors.Is(err, cobranzas.ErrCobranzaNoRevertible):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	case errors.Is(err, cobranzas.ErrAsignacionesInvalidas):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, cobranzas.ErrCobranzaDuplicateReferencia), errors.Is(err, cobranzas.ErrIdempotencyConflict):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	default:
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}

func writeReceiptPDFResponse(w http.ResponseWriter, filename string, pdf []byte) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
