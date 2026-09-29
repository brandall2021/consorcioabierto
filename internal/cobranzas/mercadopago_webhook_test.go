package cobranzas

import (
	"context"
	"strings"
	"testing"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestProcessMercadoPagoWebhookApprovesOnce(t *testing.T) {
	fake := &fakeMercadoPagoWebhookDBTX{paymentEstado: "pendiente_revision", intentStatus: "created"}
	queries := db.New(fake)

	processed, err := ProcessMercadoPagoWebhook(context.Background(), fake, queries, MercadoPagoWebhookInput{PreferenceID: "pref-1", Status: "approved"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Fatal("expected processed=true")
	}
	if fake.paymentEstado != "acreditado" {
		t.Fatalf("payment estado unexpected: %s", fake.paymentEstado)
	}
	if fake.intentStatus != "approved" {
		t.Fatalf("intent status unexpected: %s", fake.intentStatus)
	}

	processed, err = ProcessMercadoPagoWebhook(context.Background(), fake, queries, MercadoPagoWebhookInput{PreferenceID: "pref-1", Status: "approved"})
	if err != nil {
		t.Fatalf("unexpected error on repeat: %v", err)
	}
	if !processed {
		t.Fatal("expected processed=true on repeat")
	}
}

func TestProcessMercadoPagoWebhookIgnoresUnknownPreference(t *testing.T) {
	fake := &fakeMercadoPagoWebhookDBTX{missingIntent: true}
	queries := db.New(fake)

	processed, err := ProcessMercadoPagoWebhook(context.Background(), fake, queries, MercadoPagoWebhookInput{PreferenceID: "pref-missing", Status: "approved"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Fatal("expected processed=false for unknown preference")
	}
}

type fakeMercadoPagoWebhookDBTX struct {
	paymentEstado  string
	intentStatus   string
	missingIntent  bool
}

func (f *fakeMercadoPagoWebhookDBTX) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(query, "app.set_app_user_id"), strings.Contains(query, "app.set_app_tenant_id"):
		return pgconn.CommandTag{}, nil
	case strings.Contains(query, "UPDATE payments"):
		if len(args) > 0 {
			if s, ok := args[0].(string); ok {
				f.paymentEstado = s
			}
		}
		return pgconn.CommandTag{}, nil
	case strings.Contains(query, "UPDATE psp_intents"):
		if len(args) > 0 {
			if s, ok := args[0].(string); ok {
				f.intentStatus = s
			}
		}
		return pgconn.CommandTag{}, nil
	default:
		return pgconn.CommandTag{}, nil
	}
}

func (f *fakeMercadoPagoWebhookDBTX) Query(_ context.Context, query string, _ ...any) (pgx.Rows, error) {
	switch {
	case strings.Contains(query, "FROM payment_allocations") || strings.Contains(query, "FROM charges"):
		return &fakeRows{}, nil
	default:
		return nil, pgx.ErrNoRows
	}
}

func (f *fakeMercadoPagoWebhookDBTX) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	switch {
	case strings.Contains(query, "app.find_psp_intent_by_preference_id"):
		if f.missingIntent {
			return fakeRow{err: pgx.ErrNoRows}
		}
		return fakeRow{values: []any{
			stubUUID("tenant-1"),
			stubUUID("payment-1"),
			stubUUID("user-1"),
			f.paymentEstado,
			"mercadopago",
			"pref-1",
			f.intentStatus,
		}}
	case strings.Contains(query, "FROM payments"):
		return fakeRow{values: []any{
			stubUUID("tenant-1"),
			stubUUID("unit-1"),
			stubUUID("payment-1"),
			stubDate("2026-09-10"),
			"mercadopago",
			int64(12345),
			pgtype.Text{String: "", Valid: false},
			f.paymentEstado,
			"idem-1",
			pgtype.Text{String: "", Valid: false},
			stubUUID("user-1"),
			stubTime(),
			pgtype.Text{String: "mercadopago", Valid: true},
			pgtype.Text{String: "pref-1", Valid: true},
			pgtype.Text{String: "https://example.com", Valid: true},
		}}
	default:
		return fakeRow{err: pgx.ErrNoRows}
	}
}
