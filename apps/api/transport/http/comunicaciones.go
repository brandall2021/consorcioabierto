package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/comunicaciones"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *AuthHandlers) ListComunicados(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := comunicaciones.List(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"data": items, "meta": map[string]any{"request_id": middleware.GetReqID(r.Context())}})
}

func (h *AuthHandlers) CreateComunicado(w http.ResponseWriter, r *http.Request) {
	var in comunicaciones.Input
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

	item, err := comunicaciones.Create(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "comunicaciones.create", RecursoType: "comunicado", RecursoID: item.ID}))
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) PublishComunicado(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "publicar comunicados es idempotente", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := comunicaciones.Publish(r.Context(), q, chi.URLParam(r, "id"), idemKey)
	if err != nil {
		if errors.Is(err, comunicaciones.ErrIdempotencyConflict) {
			httpapi.WriteProblem(w, r, http.StatusConflict, "idempotency_conflict", "Idempotency-Key repetida", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{Accion: "comunicaciones.publish", RecursoType: "comunicado", RecursoID: item.ID}))
	httpapi.WriteJSON(w, http.StatusAccepted, item)
}
