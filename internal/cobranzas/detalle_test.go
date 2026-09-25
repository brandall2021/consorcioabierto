package cobranzas

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestGetCobranzaDetalleIncludesAsignacionesAndSaldoAFavor(t *testing.T) {
	queries := db.New(fakeDetalleDBTX{payment: stubPayment(), allocations: []db.PaymentAllocation{
		stubAllocation(2000),
		stubAllocation(1000),
	}})

	detail, err := GetCobranzaDetalle(context.Background(), queries, stubUUIDString("payment-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.SaldoAFavorCents != 2000 {
		t.Fatalf("saldo a favor unexpected: %d", detail.SaldoAFavorCents)
	}
	if len(detail.Asignaciones) != 2 {
		t.Fatalf("allocations unexpected: %d", len(detail.Asignaciones))
	}
	if detail.Cobranza.Estado != "acreditado" {
		t.Fatalf("estado unexpected: %s", detail.Cobranza.Estado)
	}
}

type fakeDetalleDBTX struct {
	payment     db.Payment
	allocations []db.PaymentAllocation
}

func (f fakeDetalleDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f fakeDetalleDBTX) Query(_ context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	switch {
	case strings.Contains(query, "FROM payment_allocations"):
		return &fakeRows{rows: f.allallocRows()}, nil
	default:
		return nil, pgx.ErrNoRows
	}
}

func (f fakeDetalleDBTX) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	switch {
	case strings.Contains(query, "FROM payments"):
		return fakeRow{values: []any{
			f.payment.TenantID,
			f.payment.UnidadID,
			f.payment.ID,
			f.payment.Fecha,
			f.payment.Canal,
			f.payment.ImporteCents,
			f.payment.Referencia,
			f.payment.Estado,
			f.payment.IdemKey,
			f.payment.MotivoRechazo,
			f.payment.CreatedBy,
			f.payment.CreatedAt,
			f.payment.PspProvider,
			f.payment.PspPreferenceID,
			f.payment.PspCheckoutUrl,
		}}
	default:
		return fakeRow{err: pgx.ErrNoRows}
	}
}

func (f fakeDetalleDBTX) allallocRows() [][]any {
	rows := make([][]any, 0, len(f.allocations))
	for _, a := range f.allocations {
		rows = append(rows, []any{a.TenantID, a.PaymentID, a.ChargeID, a.AmountCents, a.CreatedBy, a.CreatedAt})
	}
	return rows
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(r.values[i]))
	}
	return nil
}

type fakeRows struct {
	rows [][]any
	idx  int
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}
func (r *fakeRows) Values() ([]any, error) { return r.rows[r.idx-1], nil }
func (r *fakeRows) RawValues() [][]byte    { return nil }
func (r *fakeRows) Conn() *pgx.Conn        { return nil }

func stubPayment() db.Payment {
	return db.Payment{
		TenantID:     stubUUID("tenant-1"),
		UnidadID:     stubUUID("unit-1"),
		ID:           stubUUID("payment-1"),
		Fecha:        stubDate("2026-09-10"),
		Canal:        "transferencia",
		ImporteCents: 5000,
		Referencia:   stubText("ABC-123"),
		Estado:       "acreditado",
		CreatedBy:    stubUUID("user-1"),
		CreatedAt:    stubTime(),
	}
}

func stubAllocation(amount int64) db.PaymentAllocation {
	return db.PaymentAllocation{
		TenantID:    stubUUID("tenant-1"),
		PaymentID:   stubUUID("payment-1"),
		ChargeID:    stubUUID("charge-1"),
		AmountCents: amount,
		CreatedBy:   stubUUID("user-1"),
		CreatedAt:   stubTime(),
	}
}

func stubUUIDString(seed string) string {
	return stubUUID(seed).String()
}

func stubUUID(seed string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(validUUID(seed))
	return u
}

func validUUID(seed string) string {
	switch seed {
	case "tenant-1":
		return "00000000-0000-0000-0000-000000000001"
	case "unit-1":
		return "00000000-0000-0000-0000-000000000002"
	case "payment-1":
		return "00000000-0000-0000-0000-000000000003"
	case "charge-1":
		return "00000000-0000-0000-0000-000000000004"
	case "user-1":
		return "00000000-0000-0000-0000-000000000005"
	default:
		return "00000000-0000-0000-0000-000000000099"
	}
}

func stubDate(date string) pgtype.Date {
	var d pgtype.Date
	_ = d.Scan(date)
	return d
}

func stubText(value string) pgtype.Text {
	var t pgtype.Text
	_ = t.Scan(value)
	return t
}

func stubTime() pgtype.Timestamptz {
	var ts pgtype.Timestamptz
	_ = ts.Scan(time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	return ts
}
