package cobranzas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

type CheckoutInput struct {
	PaymentID   string
	UnidadID    string
	AmountCents int64
	Currency    string
	Reference   string
	BaseURL     string
}

type CheckoutResult struct {
	Provider     string `json:"provider"`
	PreferenceID string `json:"preference_id"`
	CheckoutURL  string `json:"checkout_url"`
}

type PSP interface {
	CreateCheckout(context.Context, CheckoutInput) (CheckoutResult, error)
}

func NewPSP(driver, baseURL string) PSP {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "mercadopago":
		return MercadoPagoPSP{BaseURL: baseURL}
	default:
		return MockPSP{BaseURL: baseURL}
	}
}

type MercadoPagoPSP struct {
	BaseURL string
}

func (p MercadoPagoPSP) CreateCheckout(_ context.Context, in CheckoutInput) (CheckoutResult, error) {
	pref := preferenceID("mp", in)
	return CheckoutResult{
		Provider:     "mercadopago",
		PreferenceID: pref,
		CheckoutURL:  "https://www.mercadopago.com.ar/checkout/v1/redirect?pref_id=" + pref,
	}, nil
}

type MockPSP struct {
	BaseURL string
}

func (p MockPSP) CreateCheckout(_ context.Context, in CheckoutInput) (CheckoutResult, error) {
	pref := preferenceID("mock", in)
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	return CheckoutResult{
		Provider:     "mock",
		PreferenceID: pref,
		CheckoutURL:  fmt.Sprintf("%s/api/v1/mock/checkout/%s", base, pref),
	}, nil
}

type MercadoPagoCobranzaDTO struct {
	Cobranza     CobranzaDTO `json:"cobranza"`
	CheckoutURL  string      `json:"checkout_url"`
	Provider     string      `json:"provider"`
	PreferenceID string      `json:"preference_id"`
}

func CreateMercadoPagoCobranza(ctx context.Context, q *db.Queries, consorcioID, createdBy string, in CobranzaInput, idemKey string, psp PSP) (MercadoPagoCobranzaDTO, error) {
	if psp == nil {
		psp = MockPSP{}
	}
	canal := "mercadopago"
	in.Canal = &canal
	item, err := CreateCobranza(ctx, q, consorcioID, createdBy, in, idemKey)
	if err != nil {
		return MercadoPagoCobranzaDTO{}, err
	}
	checkout, err := psp.CreateCheckout(ctx, CheckoutInput{
		PaymentID:   item.ID,
		UnidadID:    item.UnidadID,
		AmountCents: item.Importe.AmountCents,
		Currency:    item.Importe.Currency,
		Reference:   derefString(item.Referencia),
		BaseURL:     "",
	})
	if err != nil {
		return MercadoPagoCobranzaDTO{}, err
	}
	if err := q.UpdateCobranzaMercadoPago(ctx, db.UpdateCobranzaMercadoPagoParams{
		PspProvider:     pgtype.Text{String: checkout.Provider, Valid: true},
		PspPreferenceID: pgtype.Text{String: checkout.PreferenceID, Valid: true},
		PspCheckoutUrl:  pgtype.Text{String: checkout.CheckoutURL, Valid: true},
		ID:              parseUUIDMust(item.ID),
	}); err != nil {
		return MercadoPagoCobranzaDTO{}, err
	}
	if _, err := q.CreatePspIntent(ctx, db.CreatePspIntentParams{
		PaymentID:    parseUUIDMust(item.ID),
		Provider:     "mercadopago",
		PreferenceID: checkout.PreferenceID,
		Status:       "created",
	}); err != nil {
		return MercadoPagoCobranzaDTO{}, err
	}
	return MercadoPagoCobranzaDTO{
		Cobranza:     item,
		CheckoutURL:  checkout.CheckoutURL,
		Provider:     checkout.Provider,
		PreferenceID: checkout.PreferenceID,
	}, nil
}

func parseUUIDMust(s string) pgtype.UUID {
	u, _ := parseUUID(s)
	return u
}

func preferenceID(prefix string, in CheckoutInput) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		prefix,
		strings.TrimSpace(in.PaymentID),
		strings.TrimSpace(in.UnidadID),
		fmt.Sprintf("%d", in.AmountCents),
		strings.ToUpper(strings.TrimSpace(in.Currency)),
		strings.TrimSpace(in.Reference),
	}, "|")))
	return prefix + "_" + hex.EncodeToString(h[:8])
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
