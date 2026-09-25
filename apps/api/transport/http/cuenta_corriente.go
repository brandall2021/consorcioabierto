package http

import (
	"net/http"

	cc "github.com/brandall2021/consorcioabierto/internal/cuenta_corriente"
	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *AuthHandlers) GetCuentaCorriente(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	consorcioID := chi.URLParam(r, "id")
	unidadID := chi.URLParam(r, "unidadId")
	cid, err := parseUUIDValue(consorcioID)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	uid, err := parseUUIDValue(unidadID)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if _, err := q.GetUnidad(r.Context(), db.GetUnidadParams{ConsorcioID: cid, ID: uid}); err != nil {
		if err == pgx.ErrNoRows {
			httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Unidad no encontrada", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	movimientos, err := cc.ListEntriesByUnidad(r.Context(), q, unidadID)
	if err != nil {
		if err == cc.ErrCargoInvalido {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	var saldo int64
	for _, mov := range movimientos {
		saldo += mov.DebitCents - mov.CreditCents
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": movimientos,
		"meta": map[string]any{
			"request_id":  middleware.GetReqID(r.Context()),
			"saldo_cents": saldo,
		},
	})
}

func parseUUIDValue(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}
