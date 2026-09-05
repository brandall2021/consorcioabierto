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

// ListProveedores GET /consorcios/{id}/proveedores (permiso proveedores.read).
func (h *AuthHandlers) ListProveedores(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := consorcios.ListProveedores(r.Context(), q, chi.URLParam(r, "id"), consorcios.ProveedorFilter{
		Q:      r.URL.Query().Get("q"),
		Estado: r.URL.Query().Get("estado"),
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

// CreateProveedor POST /consorcios/{id}/proveedores (permiso proveedores.manage).
func (h *AuthHandlers) CreateProveedor(w http.ResponseWriter, r *http.Request) {
	var in consorcios.ProveedorInput
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

	item, err := consorcios.CreateProveedor(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		h.writeProveedorError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "proveedores.create",
		RecursoType: "proveedor",
		RecursoID:   item.ID,
		Diff:        map[string]any{"cuit": item.Cuit, "razon_social": item.RazonSocial, "consorcio_id": item.ConsorcioID},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

// GetProveedor GET /proveedores/{id} (permiso proveedores.read).
func (h *AuthHandlers) GetProveedor(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	item, err := consorcios.GetProveedor(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeProveedorError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, item)
}

// UpdateProveedor PATCH /proveedores/{id} (permiso proveedores.manage).
func (h *AuthHandlers) UpdateProveedor(w http.ResponseWriter, r *http.Request) {
	var in consorcios.ProveedorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	// UpdateProveedor exige cuit+razon_social completos; el toggle de estado usa
	// otra ruta, así que acá el body debe parsearse con esos campos presentes.
	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := consorcios.UpdateProveedor(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		h.writeProveedorError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "proveedores.update",
		RecursoType: "proveedor",
		RecursoID:   item.ID,
		Diff:        map[string]any{"cuit": item.Cuit, "razon_social": item.RazonSocial},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

// SetProveedorEstado PATCH /proveedores/{id}/estado (permiso proveedores.manage).
func (h *AuthHandlers) SetProveedorEstado(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Estado *string `json:"estado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	estado := ""
	if in.Estado != nil {
		estado = *in.Estado
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := consorcios.SetProveedorEstado(r.Context(), q, chi.URLParam(r, "id"), estado)
	if err != nil {
		h.writeProveedorError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "proveedores.update",
		RecursoType: "proveedor",
		RecursoID:   item.ID,
		Diff:        map[string]any{"estado": item.Estado},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) writeProveedorError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, consorcios.ErrProveedorInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, consorcios.ErrProveedorNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Proveedor no encontrado", err.Error(), nil)
	case errors.Is(err, consorcios.ErrProveedorDuplicateCuit):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	default:
		slog.Error("proveedores", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}