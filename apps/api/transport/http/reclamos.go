package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/reclamos"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ListReclamos GET /consorcios/{id}/reclamos (permiso reclamos.read).
func (h *AuthHandlers) ListReclamos(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := reclamos.List(r.Context(), q, chi.URLParam(r, "id"), r.URL.Query().Get("estado"))
	if err != nil {
		if errors.Is(err, reclamos.ErrInvalidID) {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

// CreateReclamo POST /consorcios/{id}/reclamos (permiso reclamos.manage).
func (h *AuthHandlers) CreateReclamo(w http.ResponseWriter, r *http.Request) {
	var in reclamos.CreateInput
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

	item, err := reclamos.Create(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "reclamos.create",
		RecursoType: "reclamo",
		RecursoID:   item.ID,
		Diff:        map[string]any{"unidad_id": item.UnidadID, "categoria": item.Categoria},
	}))
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

// GetReclamo GET /reclamos/{id} (permiso reclamos.read).
func (h *AuthHandlers) GetReclamo(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	detalle, err := reclamos.Get(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, reclamos.ErrInvalidID) {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Reclamo no encontrado", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, detalle)
}

// AddReclamoMensaje POST /reclamos/{id}/mensajes (permiso reclamos.manage).
func (h *AuthHandlers) AddReclamoMensaje(w http.ResponseWriter, r *http.Request) {
	var in reclamos.MensajeInput
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

	mensaje, err := reclamos.AddMensaje(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		if errors.Is(err, reclamos.ErrInvalidID) {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "reclamos.mensaje",
		RecursoType: "reclamo",
		RecursoID:   mensaje.ReclamoID,
	}))
	httpapi.WriteJSON(w, http.StatusCreated, mensaje)
}

// TransicionReclamo POST /reclamos/{id}/transiciones (permiso reclamos.manage).
func (h *AuthHandlers) TransicionReclamo(w http.ResponseWriter, r *http.Request) {
	var in reclamos.TransicionInput
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

	item, err := reclamos.Transicionar(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		if errors.Is(err, reclamos.ErrInvalidTransition) {
			httpapi.WriteProblem(w, r, http.StatusConflict, "invalid_transition", "Transición inválida", err.Error(), nil)
			return
		}
		if errors.Is(err, reclamos.ErrInvalidID) {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "reclamos.transicion",
		RecursoType: "reclamo",
		RecursoID:   item.ID,
		Diff:        map[string]any{"accion": in.Accion, "estado": item.Estado},
	}))
	httpapi.WriteJSON(w, http.StatusOK, item)
}
