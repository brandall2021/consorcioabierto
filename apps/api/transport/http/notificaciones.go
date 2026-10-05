package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/notificaciones"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// GetPortalNotificaciones GET /portal/notificaciones.
//
// No pide permiso nuevo, igual que el resto del portal: el scope es el vinculo
// vigente del usuario y aca mas estrecho todavia, la propia fila del usuario
// (user_id = app.current_user_id()) segun la policy de RLS de 00029. Por eso un
// consorcista no puede pedir las notificaciones de otro ni aunque lo pida.
func (h *AuthHandlers) GetPortalNotificaciones(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	soloNoLeidas, err := queryBool(r, "solo_no_leidas")
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	limite, err := queryInt(r, "limite", 0)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	listado, err := notificaciones.Listar(r.Context(), q, soloNoLeidas, limite)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": listado.Data,
		"meta": map[string]any{
			"request_id": middleware.GetReqID(r.Context()),
			"no_leidas":  listado.NoLeidas,
		},
	})
}

// MarkPortalNotificacionLeida POST /portal/notificaciones/{id}/leer.
//
// Idempotente por construccion: la query solo toca filas con leida_at IS NULL.
// Una notificacion de otro usuario no es actualizable por la policy, asi que
// responde el mismo 404 que una inexistente.
func (h *AuthHandlers) MarkPortalNotificacionLeida(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", "falta el id", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	item, err := notificaciones.MarcarLeida(r.Context(), q, id)
	if err != nil {
		if errors.Is(err, notificaciones.ErrNotificacionNotFound) {
			httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Notificación no encontrada", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, item)
}

func queryBool(r *http.Request, key string) (bool, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func queryInt(r *http.Request, key string, def int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def, nil
	}
	return strconv.Atoi(raw)
}
