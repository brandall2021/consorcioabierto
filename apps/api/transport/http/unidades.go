package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/consorcios"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ListUnidades GET /consorcios/{id}/unidades?estado= (permiso ufs.read).
func (h *AuthHandlers) ListUnidades(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := consorcios.ListUnidades(r.Context(), q, chi.URLParam(r, "id"), r.URL.Query().Get("estado"))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

// CreateUnidad POST /consorcios/{id}/unidades (permiso ufs.manage).
func (h *AuthHandlers) CreateUnidad(w http.ResponseWriter, r *http.Request) {
	var in consorcios.UnidadInput
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

	item, err := consorcios.CreateUnidad(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		h.writeUnidadError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "unidades.create",
		RecursoType: "unidad",
		RecursoID:   item.ID,
		Diff:        map[string]any{"codigo": item.Codigo, "tipo": item.Tipo, "consorcio_id": item.ConsorcioID},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) writeUnidadError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, consorcios.ErrUnidadInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, consorcios.ErrUnidadNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Unidad no encontrada", err.Error(), nil)
	case errors.Is(err, consorcios.ErrDuplicateCodigo):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	case errors.Is(err, consorcios.ErrDuplicateDocumento):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	case errors.Is(err, consorcios.ErrVinculoInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	default:
		slog.Error("unidades", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}