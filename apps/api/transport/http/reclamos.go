package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/identity"
	"github.com/brandall2021/consorcioabierto/internal/reclamos"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type reclamoInput struct {
	UnidadID  string `json:"unidad_id"`
	Categoria string `json:"categoria"`
	Texto     string `json:"texto"`
}

type reclamoMessageInput struct {
	Texto     string  `json:"texto"`
	AdjuntoID *string `json:"adjunto_id"`
}

type reclamoTransitionInput struct {
	Accion        string  `json:"accion"`
	Motivo        *string `json:"motivo"`
	ResponsableID *string `json:"responsable_id"`
}

func (h *AuthHandlers) ListReclamos(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := reclamos.ListReclamos(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeReclamoError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

func (h *AuthHandlers) GetReclamo(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	detail, err := reclamos.GetReclamoDetalle(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeReclamoError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, detail)
}

func (h *AuthHandlers) CreateReclamo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}
	var in reclamoInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if !slices.Contains(claims.Roles, "consorcista") && !h.hasPermission(r.Context(), claims, "reclamos.manage") {
		httpapi.WriteProblem(w, r, http.StatusForbidden, "forbidden", "Sin permisos", "Se requiere permiso: reclamos.manage", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := reclamos.CreateReclamo(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, reclamos.ReclamoInput(in))
	if err != nil {
		h.writeReclamoError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "reclamos.create", RecursoType: "reclamo", RecursoID: item.ID, Diff: map[string]any{"categoria": item.Categoria, "unidad_id": item.UnidadID}}))
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) AddReclamoMensaje(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}
	var in reclamoMessageInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if !slices.Contains(claims.Roles, "consorcista") && !h.hasPermission(r.Context(), claims, "reclamos.manage") {
		httpapi.WriteProblem(w, r, http.StatusForbidden, "forbidden", "Sin permisos", "Se requiere permiso: reclamos.manage", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	msg, err := reclamos.AddReclamoMensaje(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, reclamos.MessageInput(in))
	if err != nil {
		h.writeReclamoError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "reclamos.message", RecursoType: "reclamo", RecursoID: msg.ReclamoID}))
	httpapi.WriteJSON(w, http.StatusCreated, msg)
}

func (h *AuthHandlers) TransicionarReclamo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}
	if !h.hasPermission(r.Context(), claims, "reclamos.manage") {
		httpapi.WriteProblem(w, r, http.StatusForbidden, "forbidden", "Sin permisos", "Se requiere permiso: reclamos.manage", nil)
		return
	}
	var in reclamoTransitionInput
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

	item, err := reclamos.TransicionarReclamo(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, reclamos.TransitionInput(in))
	if err != nil {
		h.writeReclamoError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "reclamos.transition", RecursoType: "reclamo", RecursoID: item.ID, Diff: map[string]any{"estado": item.Estado}}))
	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) writeReclamoError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, reclamos.ErrReclamoInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, reclamos.ErrReclamoNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Reclamo no encontrado", err.Error(), nil)
	default:
		slog.Error("reclamos", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}

func (h *AuthHandlers) hasPermission(ctx context.Context, claims *identity.Claims, permission string) bool {
	perms, err := h.Manager.PermissionsForClaims(ctx, claims)
	if err != nil {
		return false
	}
	return slices.Contains(perms, permission)
}
