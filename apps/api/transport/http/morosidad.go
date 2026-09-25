package http

import (
	"net/http"

	cc "github.com/brandall2021/consorcioabierto/internal/cuenta_corriente"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *AuthHandlers) GetMorosidad(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	items, err := cc.ListMorosidadByConsorcio(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		if err == cc.ErrCargoInvalido {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
			return
		}
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	var totalSaldo int64
	var totalCargos int64
	for _, item := range items {
		totalSaldo += item.SaldoVencidoCents
		totalCargos += item.CantidadCargos
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"request_id":            middleware.GetReqID(r.Context()),
			"total_saldo_cents":     totalSaldo,
			"total_cargos_vencidos": totalCargos,
		},
	})
}
