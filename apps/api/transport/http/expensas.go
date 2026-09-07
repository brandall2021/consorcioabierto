package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/expensas"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ---- Conceptos ----

// ListConceptos GET /consorcios/{id}/conceptos (permiso expensas.read).
func (h *AuthHandlers) ListConceptos(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := expensas.ListConceptos(r.Context(), q, chi.URLParam(r, "id"), expensas.ConceptoFilter{
		Q: r.URL.Query().Get("q"),
	})
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

// CreateConcepto POST /consorcios/{id}/conceptos (permiso expensas.create).
func (h *AuthHandlers) CreateConcepto(w http.ResponseWriter, r *http.Request) {
	var in expensas.ConceptoInput
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

	item, err := expensas.CreateConcepto(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "conceptos.create",
		RecursoType: "concepto",
		RecursoID:   item.ID,
		Diff:        map[string]any{"nombre": item.Nombre, "categoria": item.Categoria, "consorcio_id": chi.URLParam(r, "id")},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

// GetConcepto GET /conceptos/{id} (permiso expensas.read).
func (h *AuthHandlers) GetConcepto(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	item, err := expensas.GetConcepto(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, item)
}

// ---- Gastos ----

// ListGastos GET /consorcios/{id}/gastos (permiso gastos.read).
func (h *AuthHandlers) ListGastos(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := expensas.ListGastos(r.Context(), q, chi.URLParam(r, "id"), expensas.GastoFilter{
		Mes:         r.URL.Query().Get("mes"),
		ProveedorID: r.URL.Query().Get("proveedor_id"),
	})
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

// CreateGasto POST /consorcios/{id}/gastos (permiso gastos.manage).
func (h *AuthHandlers) CreateGasto(w http.ResponseWriter, r *http.Request) {
	var in expensas.GastoInput
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

	item, err := expensas.CreateGasto(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "gastos.create",
		RecursoType: "gasto",
		RecursoID:   item.ID,
		Diff:        map[string]any{"concepto_id": item.ConceptoID, "importe_cents": item.ImporteCents, "fecha": item.Fecha},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

// GetGasto GET /consorcios/{id}/gastos/{gastoId} (permiso gastos.read).
func (h *AuthHandlers) GetGasto(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	item, err := expensas.GetGasto(r.Context(), q, chi.URLParam(r, "consorcioId"), chi.URLParam(r, "gastoId"))
	if err != nil {
		h.writeExpensaError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) writeExpensaError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, expensas.ErrConceptoInvalid), errors.Is(err, expensas.ErrConceptoReglaInvalid), errors.Is(err, expensas.ErrGastoInvalid),
		errors.Is(err, expensas.ErrLiquidacionInvalid), errors.Is(err, expensas.ErrLiquidacionTransicionInvalida):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, expensas.ErrConceptoNotFound), errors.Is(err, expensas.ErrGastoNotFound), errors.Is(err, expensas.ErrLiquidacionNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Recurso no encontrado", err.Error(), nil)
	case errors.Is(err, expensas.ErrConceptoDuplicate), errors.Is(err, expensas.ErrLiquidacionConflict), errors.Is(err, expensas.ErrIdempotencyConflict):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	case errors.Is(err, expensas.ErrLiquidacionVersionMismatch):
		httpapi.WriteProblem(w, r, http.StatusPreconditionFailed, "version_mismatch", "Versión no coincide", err.Error(), nil)
	case errors.Is(err, expensas.ErrLiquidacionSinGastos), errors.Is(err, expensas.ErrLiquidacionSinUFs):
		httpapi.WriteProblem(w, r, http.StatusUnprocessableEntity, "unprocessable", "No procesable", err.Error(), nil)
	default:
		slog.Error("expensas", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}
