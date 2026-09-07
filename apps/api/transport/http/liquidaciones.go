package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/expensas"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type anularLiquidacionInput struct {
	Motivo *string `json:"motivo"`
}

func (h *AuthHandlers) ListLiquidaciones(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := expensas.ListLiquidaciones(r.Context(), q, chi.URLParam(r, "id"), expensas.LiquidacionFilter{
		Periodo: r.URL.Query().Get("periodo"),
		Estado:  r.URL.Query().Get("estado"),
	})
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

func (h *AuthHandlers) GetLiquidacion(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	item, err := expensas.GetLiquidacion(r.Context(), q, chi.URLParam(r, "consorcioId"), chi.URLParam(r, "liquidacionId"))
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) CreateLiquidacion(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Periodo      string  `json:"periodo"`
		Vencimiento1 string  `json:"vencimiento_1"`
		Vencimiento2 *string `json:"vencimiento_2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := expensas.CreateLiquidacion(r.Context(), q, chi.URLParam(r, "id"), in.Periodo, in.Vencimiento1, in.Vencimiento2)
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "liquidaciones.create",
		RecursoType: "liquidacion",
		RecursoID:   item.ID,
		Diff:        map[string]any{"periodo": item.Periodo, "consorcio_id": item.ConsorcioID},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) PatchLiquidacion(w http.ResponseWriter, r *http.Request) {
	version := parseIfMatch(r)
	if version == 0 {
		httpapi.WriteProblem(w, r, http.StatusPreconditionFailed, "if_match_required", "If-Match requerido", "Se requiere header If-Match con la versión actual", nil)
		return
	}

	var in struct {
		Vencimiento1 *string `json:"vencimiento_1"`
		Vencimiento2 *string `json:"vencimiento_2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	v1 := ""
	if in.Vencimiento1 != nil {
		v1 = *in.Vencimiento1
	}
	if err := expensas.UpdateVencimientos(r.Context(), q, chi.URLParam(r, "consorcioId"), chi.URLParam(r, "liquidacionId"), v1, in.Vencimiento2, version); err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"status": "updated"})
}

func (h *AuthHandlers) CalcularLiquidacionHandler(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "El cálculo es idempotente (ADR-0004)", nil)
		return
	}
	version := parseIfMatch(r)
	if version == 0 {
		httpapi.WriteProblem(w, r, http.StatusPreconditionFailed, "if_match_required", "If-Match requerido", "Se requiere header If-Match con la versión actual", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := expensas.CalcularLiquidacion(r.Context(), q, chi.URLParam(r, "consorcioId"), chi.URLParam(r, "liquidacionId"), version, idemKey)
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "liquidaciones.calcular",
		RecursoType: "liquidacion",
		RecursoID:   item.ID,
		Diff:        map[string]any{"estado": "calculada", "periodo": item.Periodo},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) ConfirmarLiquidacionHandler(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "", nil)
		return
	}
	version := parseIfMatch(r)
	if version == 0 {
		httpapi.WriteProblem(w, r, http.StatusPreconditionFailed, "if_match_required", "If-Match requerido", "", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := expensas.ConfirmarLiquidacion(r.Context(), q, chi.URLParam(r, "consorcioId"), chi.URLParam(r, "liquidacionId"), version, idemKey)
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "liquidaciones.confirmar",
		RecursoType: "liquidacion",
		RecursoID:   item.ID,
		Diff:        map[string]any{"estado": "confirmada", "periodo": item.Periodo},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) PublicarLiquidacionHandler(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "", nil)
		return
	}
	version := parseIfMatch(r)
	if version == 0 {
		httpapi.WriteProblem(w, r, http.StatusPreconditionFailed, "if_match_required", "If-Match requerido", "", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := expensas.PublicarLiquidacion(r.Context(), q, chi.URLParam(r, "consorcioId"), chi.URLParam(r, "liquidacionId"), version, idemKey)
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "liquidaciones.publicar",
		RecursoType: "liquidacion",
		RecursoID:   item.ID,
		Diff:        map[string]any{"estado": "publicada", "periodo": item.Periodo},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) AnularLiquidacionHandler(w http.ResponseWriter, r *http.Request) {
	version := parseIfMatch(r)
	if version == 0 {
		httpapi.WriteProblem(w, r, http.StatusPreconditionFailed, "if_match_required", "If-Match requerido", "", nil)
		return
	}

	var in anularLiquidacionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	motivo := ""
	if in.Motivo != nil {
		motivo = *in.Motivo
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := expensas.AnularLiquidacion(r.Context(), q, chi.URLParam(r, "consorcioId"), chi.URLParam(r, "liquidacionId"), version, motivo)
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "liquidaciones.anular",
		RecursoType: "liquidacion",
		RecursoID:   item.ID,
		Diff:        map[string]any{"estado": "anulada", "motivo": motivo, "periodo": item.Periodo},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func parseIfMatch(r *http.Request) int {
	v := strings.TrimSpace(r.Header.Get("If-Match"))
	v = strings.TrimPrefix(v, "W/")
	v = strings.Trim(v, `"`)
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
