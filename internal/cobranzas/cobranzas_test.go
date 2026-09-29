package cobranzas

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCreateCobranzaDetectsDuplicateReferencia(t *testing.T) {
	queries := db.New(&duplicateCobranzaDBTX{})
	_, err := CreateCobranza(context.Background(), queries, "00000000-0000-0000-0000-000000000010", "00000000-0000-0000-0000-000000000020", CobranzaInput{
		UnidadID:   "00000000-0000-0000-0000-000000000030",
		Fecha:      "2026-09-28",
		Canal:      ptrStringValue("efectivo"),
		Importe:    MoneyInput{AmountCents: 10000, Currency: "ARS"},
		Referencia: ptrStringValue("REC-1"),
	}, "")
	if !errors.Is(err, ErrCobranzaDuplicateReferencia) {
		t.Fatalf("expected duplicate referencia error, got %v", err)
	}
}

func TestAcreditarCobranzaMarksPaymentAcreditado(t *testing.T) {
	fake := &creditCobranzaDBTX{}
	queries := db.New(fake)
	res, err := AcreditarCobranza(context.Background(), queries, "00000000-0000-0000-0000-000000000040", "00000000-0000-0000-0000-000000000050", nil, "idem-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Cobranza.Estado != "acreditado" {
		t.Fatalf("estado inesperado: %s", res.Cobranza.Estado)
	}
	if fake.updatedEstado != "acreditado" {
		t.Fatalf("expected payment state acreditado, got %s", fake.updatedEstado)
	}
}

func TestGetCuentaCorrienteCombinesChargesAndPayments(t *testing.T) {
	fake := &ledgerCobranzaDBTX{consorcioID: stubUUID("tenant-ledger"), unidadID: stubUUID("unit-ledger")}
	queries := db.New(fake)
	res, err := GetCuentaCorriente(context.Background(), queries, fake.consorcioID.String(), fake.unidadID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Meta.SaldoCents != 7000 {
		t.Fatalf("expected saldo 7000, got %d", res.Meta.SaldoCents)
	}
	if len(res.Data) != 2 {
		t.Fatalf("expected 2 movements, got %d", len(res.Data))
	}
	if res.Data[0].Tipo != "cargo" || res.Data[1].Tipo != "cobranza" {
		t.Fatalf("unexpected movement order: %+v", res.Data)
	}
}

func TestBuildReciboPDFStartsWithPDF(t *testing.T) {
	pdf := BuildReciboPDF(AcreditacionDTO{Cobranza: CobranzaDTO{ID: "cob-1", UnidadID: "uni-1", Fecha: "2026-09-28", Canal: "efectivo", Importe: MoneyDTO{AmountCents: 10000, Currency: "ARS"}}, Asignaciones: []AsignacionDTO{{ChargeID: "chg-1", AmountCents: 4000}}, SaldoAFavorCents: 6000})
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("expected pdf header, got %q", pdf[:8])
	}
}

func TestAjustarAsignacionesReemplazaAllocations(t *testing.T) {
	fake := &adjustCobranzaDBTX{}
	queries := db.New(fake)
	res, err := AjustarAsignaciones(context.Background(), queries, "00000000-0000-0000-0000-000000000060", "00000000-0000-0000-0000-000000000070", []AsignacionInput{{
		ChargeID:    "00000000-0000-0000-0000-000000000080",
		AmountCents: 4000,
	}}, "idem-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fake.deleted {
		t.Fatalf("expected allocations to be replaced")
	}
	if fake.updatedEstado != "acreditado" {
		t.Fatalf("expected payment state acreditado, got %s", fake.updatedEstado)
	}
	if res.SaldoAFavorCents != 6000 {
		t.Fatalf("expected saldo a favor 6000, got %d", res.SaldoAFavorCents)
	}
	if len(res.Asignaciones) != 1 || res.Asignaciones[0].AmountCents != 4000 {
		t.Fatalf("unexpected allocations: %+v", res.Asignaciones)
	}
}

type duplicateCobranzaDBTX struct{ calls int }

func (f *duplicateCobranzaDBTX) Exec(_ context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	q := strings.ToLower(query)
	if strings.Contains(q, "insert into idempotency_keys") {
		return pgconn.CommandTag{}, nil
	}
	if strings.Contains(q, "insert into payments") {
		return pgconn.CommandTag{}, &pgconn.PgError{Code: "23505", ConstraintName: "payments_referencia_unique_idx"}
	}
	return pgconn.CommandTag{}, nil
}

func (f *duplicateCobranzaDBTX) Query(_ context.Context, query string, _ ...interface{}) (pgx.Rows, error) {
	if strings.Contains(query, "FROM charges") || strings.Contains(query, "FROM payment_allocations") {
		return &fakeRows{}, nil
	}
	return &fakeRows{}, nil
}

func (f *duplicateCobranzaDBTX) QueryRow(_ context.Context, query string, _ ...interface{}) pgx.Row {
	f.calls++
	switch f.calls {
	case 1:
		return fakeRow{values: []any{
			stubUUID("tenant-dup"),
			stubUUID("consorcio-dup"),
			stubUUID("unit-dup"),
			"A-101",
			"departamento",
			stubNumeric("80"),
			stubNumeric("1"),
			"activo",
			stubTime(),
			stubTime(),
		}}
	case 2:
		return fakeRow{err: &pgconn.PgError{Code: "23505", ConstraintName: "payments_referencia_unique_idx"}}
	default:
		return fakeRow{err: pgx.ErrNoRows}
	}
}

type creditCobranzaDBTX struct{ updatedEstado string }

func (f *creditCobranzaDBTX) Exec(_ context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if strings.Contains(query, "UPDATE payments") && len(args) >= 1 {
		if s, ok := args[0].(string); ok {
			f.updatedEstado = s
		}
	}
	return pgconn.CommandTag{}, nil
}

func (f *creditCobranzaDBTX) Query(_ context.Context, query string, _ ...interface{}) (pgx.Rows, error) {
	if strings.Contains(query, "FROM payment_allocations") || strings.Contains(query, "FROM charges") {
		return &fakeRows{}, nil
	}
	return &fakeRows{}, nil
}

func (f *creditCobranzaDBTX) QueryRow(_ context.Context, query string, _ ...interface{}) pgx.Row {
	switch {
	case strings.Contains(query, "FROM payments"):
		estado := f.updatedEstado
		if estado == "" {
			estado = "pendiente_revision"
		}
		return fakeRow{values: []any{
			stubUUID("tenant-pay"),
			stubUUID("unit-pay"),
			stubUUID("payment-pay"),
			stubDate("2026-09-28"),
			"efectivo",
			int64(10000),
			pgtype.Text{},
			estado,
			"idem-pay",
			pgtype.Text{},
			stubUUID("user-pay"),
			stubTime(),
			pgtype.Text{},
			pgtype.Text{},
			pgtype.Text{},
		}}
	default:
		return fakeRow{err: pgx.ErrNoRows}
	}
}

type adjustCobranzaDBTX struct {
	updatedEstado string
	deleted       bool
	calls         int
}

type ledgerCobranzaDBTX struct {
	consorcioID pgtype.UUID
	unidadID    pgtype.UUID
}

func (f *ledgerCobranzaDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *ledgerCobranzaDBTX) Query(_ context.Context, query string, _ ...interface{}) (pgx.Rows, error) {
	if strings.Contains(query, "FROM charges") {
		return &fakeRows{rows: []fakeRow{{values: []any{
			stubUUID("tenant-ledger"),
			f.unidadID,
			stubUUID("charge-ledger"),
			stubUUID("liquidacion-ledger"),
			"Expensa octubre",
			stubDate("2026-09-01"),
			int64(10000),
			int64(10000),
			stubTime(),
		}}}}, nil
	}
	if strings.Contains(query, "FROM payments") {
		return &fakeRows{rows: []fakeRow{{values: []any{
			stubUUID("tenant-ledger"),
			f.unidadID,
			stubUUID("payment-ledger"),
			stubDate("2026-09-10"),
			"efectivo",
			int64(3000),
			pgtype.Text{},
			"acreditado",
			"idem-ledger",
			pgtype.Text{},
			stubUUID("user-ledger"),
			stubTime(),
			pgtype.Text{},
			pgtype.Text{},
			pgtype.Text{},
		}}}}, nil
	}
	return &fakeRows{}, nil
}

func (f *ledgerCobranzaDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakeRow{err: pgx.ErrNoRows}
}

func (f *adjustCobranzaDBTX) Exec(_ context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if strings.Contains(query, "DELETE FROM payment_allocations") {
		f.deleted = true
	}
	if strings.Contains(query, "UPDATE payments") && len(args) >= 1 {
		if s, ok := args[0].(string); ok {
			f.updatedEstado = s
		}
	}
	return pgconn.CommandTag{}, nil
}

func (f *adjustCobranzaDBTX) Query(_ context.Context, query string, _ ...interface{}) (pgx.Rows, error) {
	return &fakeRows{}, nil
}

func (f *adjustCobranzaDBTX) QueryRow(_ context.Context, query string, _ ...interface{}) pgx.Row {
	f.calls++
	switch f.calls {
	case 1:
		return fakeRow{values: []any{
			stubUUID("tenant-adjust"),
			stubUUID("unit-adjust"),
			stubUUID("payment-adjust"),
			stubDate("2026-09-28"),
			"efectivo",
			int64(10000),
			pgtype.Text{},
			"pendiente_revision",
			"idem-adjust",
			pgtype.Text{},
			stubUUID("user-adjust"),
			stubTime(),
			pgtype.Text{},
			pgtype.Text{},
			pgtype.Text{},
		}}
	case 2:
		return fakeRow{values: []any{
			stubUUID("tenant-adjust"),
			stubUUID("payment-adjust"),
			stubUUID("charge-adjust"),
			int64(4000),
			stubUUID("user-adjust"),
			stubTime(),
		}}
	default:
		return fakeRow{err: pgx.ErrNoRows}
	}
}

func ptrStringValue(s string) *string { return &s }
