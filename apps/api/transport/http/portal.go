package http

import (
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/portal"
	"github.com/go-chi/chi/v5/middleware"
)

// GetPortalHome GET /portal.
func (h *AuthHandlers) GetPortalHome(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	home, err := portal.BuildTenantHome(r.Context(), q)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": home,
		"meta": map[string]any{"request_id": middleware.GetReqID(r.Context())},
	})
}
