package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/consorcios"
	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/tenancy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// txQueries abre una transacción con el contexto de RLS del tenant activo y
// devuelve un *db.Queries acotado. El caller decide commit o rollback.
func (h *AuthHandlers) txQueries(r *http.Request) (*db.Queries, func() error, func() error, error) {
	ctx := r.Context()
	claims := claimsFrom(ctx)
	if claims == nil {
		return nil, nil, nil, errors.New("claims ausentes")
	}
	tx, err := h.Manager.Pool().Begin(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := tenancy.SetContext(ctx, tx, claims.Subject, claims.Tenant); err != nil {
		_ = tx.Rollback(ctx)
		return nil, nil, nil, err
	}
	commit := func() error { return tx.Commit(ctx) }
	rollback := func() error { return tx.Rollback(ctx) }
	return db.New(tx), commit, rollback, nil
}

// rollbackOnly es un helper para handlers de solo lectura (commit innecesario).
func rollbackOnly(rollback func() error) func() { return func() { _ = rollback() } }

// consorcioAuditEvent arma un evento de auditoría con el tenant y la membresía
// del token (las acciones de consorcios ocurren dentro del tenant activo).
func (h *AuthHandlers) consorcioAuditEvent(r *http.Request, e audit.Event) audit.Event {
	if c := claimsFrom(r.Context()); c != nil {
		e.TenantID = c.Tenant
		e.ActorMembership = c.Membership
	}
	return e
}

// ListConsorcios GET /consorcios?q=&estado= (permiso consorcios.read).
func (h *AuthHandlers) ListConsorcios(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := consorcios.List(r.Context(), q, consorcios.Filter{
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

// CreateConsorcio POST /consorcios (permiso consorcios.manage).
func (h *AuthHandlers) CreateConsorcio(w http.ResponseWriter, r *http.Request) {
	var in consorcios.Input
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

	item, err := consorcios.Create(r.Context(), q, in)
	if err != nil {
		h.writeConsorcioError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "consorcios.create",
		RecursoType: "consorcio",
		RecursoID:   item.ID,
		Diff:        map[string]any{"nombre": item.Nombre, "tipo": item.Tipo},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, item)
}

// GetConsorcio GET /consorcios/{id} (permiso consorcios.read).
func (h *AuthHandlers) GetConsorcio(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	item, err := consorcios.Get(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeConsorcioError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, item)
}

// UpdateConsorcio PATCH /consorcios/{id} (permiso consorcios.manage).
func (h *AuthHandlers) UpdateConsorcio(w http.ResponseWriter, r *http.Request) {
	var in consorcios.Input
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

	item, err := consorcios.Update(r.Context(), q, chi.URLParam(r, "id"), in)
	if err != nil {
		h.writeConsorcioError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "consorcios.update",
		RecursoType: "consorcio",
		RecursoID:   item.ID,
		Diff:        map[string]any{"nombre": item.Nombre, "estado": item.Estado},
	}))

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *AuthHandlers) writeConsorcioError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, consorcios.ErrInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, consorcios.ErrNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Consorcio no encontrado", err.Error(), nil)
	case errors.Is(err, consorcios.ErrDuplicateName):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	default:
		slog.Error("consorcios", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}
