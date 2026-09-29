package http

import (
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	portalhome "github.com/brandall2021/consorcioabierto/internal/portal"
	"github.com/go-chi/chi/v5/middleware"
)

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
