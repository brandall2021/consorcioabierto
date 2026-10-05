package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/personas"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// MiembroDTO en el contrato HTTP: solo identidad y estado, nunca el hash de
// contraseña ni el secreto MFA.
type MiembroDTO = personas.MiembroDTO

// PersonaVinculoDTO en el contrato HTTP.
type PersonaVinculoDTO = personas.PersonaVinculoDTO

type vincularUsuarioRequest struct {
	UsuarioID string `json:"usuario_id"`
}

// ListTenantMembers GET /tenant/members (permiso tenant.users.read).
// Expone únicamente los miembros del tenant activo: el filtro por tenant lo
// aplica la vista app.v_tenant_members, no el handler.
func (h *AuthHandlers) ListTenantMembers(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := personas.ListarMiembros(r.Context(), q)
	if err != nil {
		slog.Error("personas", "op", "listar_miembros", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

// VincularPersonaUsuario PUT /personas/{personaId}/usuario (permiso tenant.users.manage).
// Es la operación que cierra el circuito del portal: sin este vínculo el
// consorcista no tiene unidades y ve un portal vacío.
func (h *AuthHandlers) VincularPersonaUsuario(w http.ResponseWriter, r *http.Request) {
	var in vincularUsuarioRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if strings.TrimSpace(in.UsuarioID) == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", "usuario_id es obligatorio", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	personaID := chi.URLParam(r, "personaId")
	item, err := personas.Vincular(r.Context(), q, personaID, in.UsuarioID)
	if err != nil {
		h.writePersonaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "personas.usuario.vincular",
		RecursoType: "persona",
		RecursoID:   item.ID,
		Diff:        map[string]any{"usuario_id": in.UsuarioID},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

// DesvincularPersonaUsuario DELETE /personas/{personaId}/usuario (permiso tenant.users.manage).
func (h *AuthHandlers) DesvincularPersonaUsuario(w http.ResponseWriter, r *http.Request) {
	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	personaID := chi.URLParam(r, "personaId")
	item, err := personas.Desvincular(r.Context(), q, personaID)
	if err != nil {
		h.writePersonaError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "personas.usuario.desvincular",
		RecursoType: "persona",
		RecursoID:   item.ID,
		Diff:        map[string]any{"usuario_id": nil},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) writePersonaError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, personas.ErrVinculoInvalido):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, personas.ErrPersonaNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Persona no encontrada", err.Error(), nil)
	case errors.Is(err, personas.ErrUsuarioNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Usuario no encontrado", err.Error(), nil)
	case errors.Is(err, personas.ErrUsuarioYaVinculado):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	default:
		slog.Error("personas", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}
