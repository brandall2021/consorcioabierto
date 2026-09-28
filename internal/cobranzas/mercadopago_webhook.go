package cobranzas

import (
	"context"
	"errors"
	"fmt"
	"strings"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type MercadoPagoWebhookInput struct {
	PreferenceID string
	Status       string
}

type mercadoPagoIntentLookup struct {
	TenantID     pgtype.UUID
	PaymentID    pgtype.UUID
	CreatedBy    pgtype.UUID
	PaymentState string
	Provider     string
	PreferenceID string
	Status       string
}

const lookupMercadoPagoIntentSQL = `
SELECT tenant_id, payment_id, created_by, payment_estado, provider, preference_id, status
FROM app.find_psp_intent_by_preference_id($1::TEXT)
`

func ProcessMercadoPagoWebhook(ctx context.Context, tx db.DBTX, q *db.Queries, in MercadoPagoWebhookInput) (bool, error) {
	pref := strings.TrimSpace(in.PreferenceID)
	if pref == "" {
		return false, ErrCobranzaInvalid
	}

	intent, err := lookupMercadoPagoIntent(ctx, tx, pref)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := setRLSForWebhook(ctx, tx, intent.CreatedBy.String(), intent.TenantID.String()); err != nil {
		return false, err
	}

	status := strings.ToLower(strings.TrimSpace(in.Status))
	switch status {
	case "approved":
		payment, err := q.GetCobranza(ctx, intent.PaymentID)
		if err != nil {
			return false, err
		}
		if payment.Estado == "pendiente_revision" {
			if _, err := AcreditarCobranza(ctx, q, intent.PaymentID.String(), intent.CreatedBy.String(), nil, ""); err != nil {
				return false, err
			}
		}
	case "pending", "rejected", "cancelled", "authorized":
		// Solo persistimos el estado del intento.
	default:
		// Estados desconocidos se registran sin romper el webhook.
	}

	if err := q.UpdatePspIntentStatus(ctx, db.UpdatePspIntentStatusParams{
		PreferenceID: pref,
		Status:       status,
	}); err != nil {
		return false, err
	}

	return status == "approved", nil
}

func lookupMercadoPagoIntent(ctx context.Context, tx db.DBTX, preferenceID string) (mercadoPagoIntentLookup, error) {
	row := tx.QueryRow(ctx, lookupMercadoPagoIntentSQL, preferenceID)
	var out mercadoPagoIntentLookup
	if err := row.Scan(&out.TenantID, &out.PaymentID, &out.CreatedBy, &out.PaymentState, &out.Provider, &out.PreferenceID, &out.Status); err != nil {
		return mercadoPagoIntentLookup{}, err
	}
	return out, nil
}

func setRLSForWebhook(ctx context.Context, tx db.DBTX, userID, tenantID string) error {
	if err := setRLSValue(ctx, tx, "app.set_app_user_id", userID); err != nil {
		return err
	}
	return setRLSValue(ctx, tx, "app.set_app_tenant_id", tenantID)
}

func setRLSValue(ctx context.Context, tx db.DBTX, fn, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s requerido", fn)
	}
	_, err := tx.Exec(ctx, "SELECT "+fn+"($1)", value)
	return err
}
