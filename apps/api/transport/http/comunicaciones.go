package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/comunicaciones"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type comunicadoInput struct {
	Titulo        string `json:"titulo"`
	Cuerpo        string `json:"cuerpo"`
	Destinatarios string `json:"destinatarios"`
}

func (h *AuthHandlers) ListComunicados(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := comunicaciones.ListComunicados(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeComunicadoError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}

func (h *AuthHandlers) CreateComunicado(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}
	var in comunicadoInput
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

	item, err := comunicaciones.CreateComunicado(r.Context(), q, chi.URLParam(r, "id"), claims.Subject, comunicaciones.ComunicadoInput(in))
	if err != nil {
		h.writeComunicadoError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "comunicaciones.create", RecursoType: "comunicado", RecursoID: item.ID, Diff: map[string]any{"titulo": item.Titulo, "destinatarios": item.Destinatarios}}))
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) PublicarComunicado(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Idempotency-Key") == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "La publicación es idempotente", nil)
		return
	}
	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := comunicaciones.PublicarComunicado(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeComunicadoError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "comunicaciones.publish", RecursoType: "comunicado", RecursoID: item.ID, Diff: map[string]any{"estado": item.Estado}}))
	httpapi.WriteJSON(w, http.StatusAccepted, item)
}

func (h *AuthHandlers) writeComunicadoError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, comunicaciones.ErrComunicadoInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, comunicaciones.ErrComunicadoNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Comunicado no encontrado", err.Error(), nil)
	case errors.Is(err, comunicaciones.ErrComunicadoPublished):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Comunicado publicado", err.Error(), nil)
	default:
		slog.Error("comunicaciones", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}
