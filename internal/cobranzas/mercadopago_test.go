package cobranzas

import (
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCreateMercadoPagoCobranzaBuildsCheckoutURL(t *testing.T) {
	queries := db.New(&fakeMercadoPagoDBTX{})
	psp := NewPSP("mercadopago", "https://app.example")

	res, err := CreateMercadoPagoCobranza(context.Background(), queries, "00000000-0000-0000-0000-000000000010", "00000000-0000-0000-0000-000000000020", CobranzaInput{
		UnidadID: "00000000-0000-0000-0000-000000000030",
		Fecha:    "2026-09-22",
		Importe:  MoneyInput{AmountCents: 12345, Currency: "ARS"},
	}, "", psp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Cobranza.Canal != "mercadopago" {
		t.Fatalf("channel unexpected: %s", res.Cobranza.Canal)
	}
	if !strings.Contains(res.CheckoutURL, "pref_id=") {
		t.Fatalf("checkout url unexpected: %s", res.CheckoutURL)
	}
	if res.Provider != "mercadopago" {
		t.Fatalf("provider unexpected: %s", res.Provider)
	}
}

func TestNewPSPMercadoPagoAndMockAreDeterministic(t *testing.T) {
	input := CheckoutInput{PaymentID: "payment-1", AmountCents: 5000, Currency: "ARS"}

	for _, tc := range []struct {
		name   string
		driver string
		url    string
	}{
		{name: "mercadopago", driver: "mercadopago", url: "https://www.mercadopago.com.ar/checkout/v1/redirect?pref_id="},
		{name: "mock", driver: "mock", url: "https://app.example/api/v1/mock/checkout/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			psp := NewPSP(tc.driver, "https://app.example")
			res, err := psp.CreateCheckout(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasPrefix(res.CheckoutURL, tc.url) {
				t.Fatalf("checkout url unexpected: %s", res.CheckoutURL)
			}
			if res.PreferenceID == "" {
				t.Fatal("expected preference id")
			}
		})
	}
}

type fakeMercadoPagoDBTX struct{}

func (f *fakeMercadoPagoDBTX) Exec(_ context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if strings.Contains(query, "UPDATE payments") {
		if len(args) != 4 {
			panic("unexpected args")
		}
		return pgconn.CommandTag{}, nil
	}
	return pgconn.CommandTag{}, nil
}

func (f *fakeMercadoPagoDBTX) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return &fakeRows{}, nil
}

func (f *fakeMercadoPagoDBTX) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	switch {
	case strings.Contains(query, "FROM unidades"):
		return fakeRow{values: []any{
			stubUUID("tenant-1"),
			stubUUID("consorcio-1"),
			stubUUID("unit-1"),
			"A-101",
			"departamento",
			stubNumeric("80"),
			stubNumeric("1"),
			"activo",
			stubTime(),
			stubTime(),
		}}
	case strings.Contains(query, "INSERT INTO payments"):
		return fakeRow{values: []any{
			stubUUID("tenant-1"),
			stubUUID("unit-1"),
			stubUUID("payment-1"),
			stubDate("2026-09-22"),
			"mercadopago",
			int64(12345),
			pgtype.Text{String: "", Valid: false},
			"pendiente_revision",
			"idem-mercadopago",
			pgtype.Text{String: "", Valid: false},
			stubUUID("user-1"),
			stubTime(),
			pgtype.Text{String: "", Valid: false},
			pgtype.Text{String: "", Valid: false},
			pgtype.Text{String: "", Valid: false},
		}}
	case strings.Contains(query, "INSERT INTO psp_intents"):
		return fakeRow{values: []any{
			stubUUID("tenant-1"),
			stubUUID("intent-1"),
			stubUUID("payment-1"),
			"mercadopago",
			"pref-1",
			"created",
			stubTime(),
			stubTime(),
		}}
	default:
		return fakeRow{err: pgx.ErrNoRows}
	}
}

func stubNumeric(value string) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(value)
	return n
}

func TestCreateMercadoPagoCobranzaRejectsInvalidInput(t *testing.T) {
	if _, err := CreateMercadoPagoCobranza(context.Background(), db.New(&fakeMercadoPagoDBTX{}), "bad", "user", CobranzaInput{}, "", NewPSP("mercadopago", "https://app.example")); !errors.Is(err, ErrCobranzaInvalid) {
		t.Fatalf("expected ErrCobranzaInvalid, got %v", err)
	}
}
