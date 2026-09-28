package http

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/brandall2021/consorcioabierto/internal/cobranzas"
	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
)

const mercadoPagoWebhookSecretHeader = "X-Mercado-Pago-Webhook-Secret"

type mercadoPagoWebhookRequest struct {
	PreferenceID string `json:"preference_id"`
	Status       string `json:"status"`
}

func (h *AuthHandlers) MercadoPagoWebhook(w http.ResponseWriter, r *http.Request) {
	var req mercadoPagoWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	if !validateMercadoPagoWebhookSecret(h.MercadoPagoWebhookSecret, r.Header.Get(mercadoPagoWebhookSecretHeader)) {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Webhook inválido", "secret inválido", nil)
		return
	}

	tx, err := h.Manager.Pool().Begin(r.Context())
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	q := db.New(tx)
	processed, err := cobranzas.ProcessMercadoPagoWebhook(r.Context(), tx, q, cobranzas.MercadoPagoWebhookInput{
		PreferenceID: req.PreferenceID,
		Status:       req.Status,
	})
	if err != nil {
		httpapi.WriteProblem(w, r, webHookStatusFromError(err), "invalid_request", "Webhook inválido", err.Error(), nil)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"processed": processed})
}

func validateMercadoPagoWebhookSecret(expected, provided string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	provided = strings.TrimSpace(provided)
	if len(expected) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func webHookStatusFromError(err error) int {
	if err == cobranzas.ErrCobranzaInvalid {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
