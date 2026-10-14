package cobranzas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
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

func NewPSP(driver, baseURL, accessToken string) PSP {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "mercadopago":
		return MercadoPagoPSP{AppBaseURL: baseURL, AccessToken: accessToken}
	default:
		return MockPSP{BaseURL: baseURL}
	}
}

type MercadoPagoPSP struct {
	AppBaseURL  string
	APIBaseURL  string
	AccessToken string
	HTTPClient  HTTPDoer
}

type HTTPDoer interface {
	Do(*nethttp.Request) (*nethttp.Response, error)
}

type mercadoPagoPreferenceRequest struct {
	ExternalReference string            `json:"external_reference"`
	NotificationURL   string            `json:"notification_url,omitempty"`
	AutoReturn        string            `json:"auto_return,omitempty"`
	Items             []mercadoPagoItem `json:"items"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	BackURLs          map[string]string `json:"back_urls,omitempty"`
	BinaryMode        bool              `json:"binary_mode,omitempty"`
}

type mercadoPagoItem struct {
	Title     string  `json:"title"`
	Quantity  int     `json:"quantity"`
	Currency  string  `json:"currency_id"`
	UnitPrice float64 `json:"unit_price"`
}

type mercadoPagoPreferenceResponse struct {
	ID               string `json:"id"`
	InitPoint        string `json:"init_point"`
	SandboxInitPoint string `json:"sandbox_init_point"`
}

func (p MercadoPagoPSP) CreateCheckout(ctx context.Context, in CheckoutInput) (CheckoutResult, error) {
	if strings.TrimSpace(p.AccessToken) == "" {
		return CheckoutResult{}, errors.New("MERCADO_PAGO_ACCESS_TOKEN requerido")
	}
	client := p.HTTPClient
	if client == nil {
		client = nethttp.DefaultClient
	}
	apiBase := strings.TrimRight(strings.TrimSpace(p.APIBaseURL), "/")
	if apiBase == "" {
		apiBase = "https://api.mercadopago.com"
	}
	appBase := strings.TrimRight(strings.TrimSpace(p.AppBaseURL), "/")
	prefReq := mercadoPagoPreferenceRequest{
		ExternalReference: strings.TrimSpace(in.PaymentID),
		NotificationURL:   appBase + "/api/v1/webhooks/mercado-pago",
		AutoReturn:        "approved",
		Items: []mercadoPagoItem{{
			Title:     "ConsorcioAbierto - cobranza",
			Quantity:  1,
			Currency:  strings.ToUpper(strings.TrimSpace(in.Currency)),
			UnitPrice: float64(in.AmountCents) / 100,
		}},
		Metadata: map[string]string{
			"payment_id": strings.TrimSpace(in.PaymentID),
			"unidad_id":  strings.TrimSpace(in.UnidadID),
		},
		BackURLs: map[string]string{
			"success": appBase + "/app/consorcios",
			"pending": appBase + "/app/consorcios",
			"failure": appBase + "/app/consorcios",
		},
		BinaryMode: true,
	}
	body, err := json.Marshal(prefReq)
	if err != nil {
		return CheckoutResult{}, err
	}
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodPost, apiBase+"/checkout/preferences", strings.NewReader(string(body)))
	if err != nil {
		return CheckoutResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return CheckoutResult{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CheckoutResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CheckoutResult{}, fmt.Errorf("mercado pago create preference: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var out mercadoPagoPreferenceResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return CheckoutResult{}, err
	}
	checkoutURL := strings.TrimSpace(out.InitPoint)
	if checkoutURL == "" {
		checkoutURL = strings.TrimSpace(out.SandboxInitPoint)
	}
	if checkoutURL == "" {
		return CheckoutResult{}, errors.New("mercado pago checkout_url ausente")
	}
	return CheckoutResult{Provider: "mercadopago", PreferenceID: strings.TrimSpace(out.ID), CheckoutURL: checkoutURL}, nil
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
