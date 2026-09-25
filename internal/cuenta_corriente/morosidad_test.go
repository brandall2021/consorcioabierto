package cuenta_corriente

import (
	"context"
	"reflect"
	"testing"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestListMorosidadByConsorcioReturnsUnitSummaries(t *testing.T) {
	queries := db.New(fakeMorosidadDBTX{rows: [][]any{
		{stubUUID("unit-1"), "UF 1", int64(8000), int64(2), stubDate("2026-08-15")},
		{stubUUID("unit-2"), "UF 2", int64(3500), int64(1), stubDate("2026-09-01")},
	}})

	items, err := ListMorosidadByConsorcio(context.Background(), queries, stubUUIDStringMorosidad("consorcio-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].UnidadCodigo != "UF 1" || items[0].SaldoVencidoCents != 8000 || items[0].CantidadCargos != 2 {
		t.Fatalf("first item unexpected: %+v", items[0])
	}
	if items[1].UnidadCodigo != "UF 2" || items[1].SaldoVencidoCents != 3500 || items[1].CantidadCargos != 1 {
		t.Fatalf("second item unexpected: %+v", items[1])
	}
}

type fakeMorosidadDBTX struct {
	rows [][]any
}

func (f fakeMorosidadDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f fakeMorosidadDBTX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return &fakeMorosidadRows{rows: f.rows}, nil
}

func (f fakeMorosidadDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakeMorosidadRow{err: pgx.ErrNoRows}
}

type fakeMorosidadRow struct {
	err error
}

func (r fakeMorosidadRow) Scan(dest ...any) error { return r.err }

type fakeMorosidadRows struct {
	rows [][]any
	idx  int
}

func (r *fakeMorosidadRows) Close()                                       {}
func (r *fakeMorosidadRows) Err() error                                   { return nil }
func (r *fakeMorosidadRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeMorosidadRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeMorosidadRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeMorosidadRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}
func (r *fakeMorosidadRows) Values() ([]any, error) { return r.rows[r.idx-1], nil }
func (r *fakeMorosidadRows) RawValues() [][]byte    { return nil }
func (r *fakeMorosidadRows) Conn() *pgx.Conn        { return nil }

func stubUUIDStringMorosidad(seed string) string { return stubUUIDMorosidad(seed).String() }

func stubUUIDMorosidad(seed string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(validMorosidadUUID(seed))
	return u
}

func validMorosidadUUID(seed string) string {
	switch seed {
	case "consorcio-1":
		return "00000000-0000-0000-0000-000000000011"
	case "unit-1":
		return "00000000-0000-0000-0000-000000000012"
	case "unit-2":
		return "00000000-0000-0000-0000-000000000013"
	default:
		return "00000000-0000-0000-0000-000000000099"
	}
}
