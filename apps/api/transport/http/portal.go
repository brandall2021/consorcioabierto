package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	portalhome "github.com/brandall2021/consorcioabierto/internal/portal"
	"github.com/brandall2021/consorcioabierto/internal/reclamos"
	"github.com/go-chi/chi/v5/middleware"
)

// CreatePortalReclamo POST /portal/reclamos.
//
// No pide permiso nuevo: lo que acota es el vinculo vigente del usuario, que
// el servidor resuelve con ListUnidadesForCurrentUser. El consorcio sale de la
// unidad resuelta, nunca del cuerpo del request.
func (h *AuthHandlers) CreatePortalReclamo(w http.ResponseWriter, r *http.Request) {
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

	item, err := portalhome.CrearReclamo(r.Context(), q, q, in)
	if err != nil {
		if errors.Is(err, portalhome.ErrSinVinculo) {
			httpapi.WriteProblem(w, r, http.StatusForbidden, "forbidden", "Sin vínculo con esa unidad", err.Error(), nil)
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
		Accion:      "reclamos.create",
		RecursoType: "reclamo",
		RecursoID:   item.ID,
		Diff:        map[string]any{"unidad_id": item.UnidadID, "categoria": item.Categoria, "origen": "portal"},
	}))
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *AuthHandlers) GetPortalHome(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	home, err := portalhome.BuildHome(r.Context(), q)
	if err != nil {
		slog.Error("portal home", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": home,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}
