package cuenta_corriente

import (
	"context"
	"errors"
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

func TestListMorosidadByConsorcioForwardsScopeAndReturnsOnlyScopedRows(t *testing.T) {
	const (
		scopeA = "00000000-0000-0000-0000-000000000011"
		scopeB = "00000000-0000-0000-0000-000000000014"
	)
	fake := &scopedMorosidadDBTX{rowsByScope: map[string][][]any{
		scopeA: {{stubUUID("unit-1"), "UF 1", int64(8000), int64(2), stubDate("2026-08-15")}},
		scopeB: {{stubUUID("unit-2"), "UF 2", int64(3500), int64(1), stubDate("2026-09-01")}},
	}}
	queries := db.New(fake)

	itemsA, err := ListMorosidadByConsorcio(context.Background(), queries, scopeA)
	if err != nil {
		t.Fatalf("unexpected error for scope A: %v", err)
	}
	if len(fake.calls) != 1 || fake.calls[0] != scopeA {
		t.Fatalf("expected scope A forwarded to the query, got calls %v", fake.calls)
	}
	if len(itemsA) != 1 || itemsA[0].UnidadCodigo != "UF 1" || itemsA[0].SaldoVencidoCents != 8000 {
		t.Fatalf("scope A expected only its own rows, got %+v", itemsA)
	}

	itemsB, err := ListMorosidadByConsorcio(context.Background(), queries, scopeB)
	if err != nil {
		t.Fatalf("unexpected error for scope B: %v", err)
	}
	if len(fake.calls) != 2 || fake.calls[1] != scopeB {
		t.Fatalf("expected scope B forwarded to the query, got calls %v", fake.calls)
	}
	if len(itemsB) != 1 || itemsB[0].UnidadCodigo != "UF 2" || itemsB[0].SaldoVencidoCents != 3500 {
		t.Fatalf("scope B expected only its own rows, got %+v", itemsB)
	}

	for _, item := range itemsA {
		if item.UnidadCodigo == "UF 2" {
			t.Fatalf("scope B row leaked into scope A result: %+v", item)
		}
	}
	for _, item := range itemsB {
		if item.UnidadCodigo == "UF 1" {
			t.Fatalf("scope A row leaked into scope B result: %+v", item)
		}
	}
}

func TestListMorosidadByConsorcioInvalidScopeDoesNotQuery(t *testing.T) {
	fake := &scopedMorosidadDBTX{rowsByScope: map[string][][]any{}}
	queries := db.New(fake)

	_, err := ListMorosidadByConsorcio(context.Background(), queries, "no-es-uuid")
	if !errors.Is(err, ErrCargoInvalido) {
		t.Fatalf("expected ErrCargoInvalido, got %v", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("expected no query without a valid scope, got calls %v", fake.calls)
	}
}

type scopedMorosidadDBTX struct {
	rowsByScope map[string][][]any
	calls       []string
}

func (f *scopedMorosidadDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *scopedMorosidadDBTX) Query(_ context.Context, _ string, args ...interface{}) (pgx.Rows, error) {
	scope := scopedQueryKey(args)
	f.calls = append(f.calls, scope)
	return &fakeMorosidadRows{rows: f.rowsByScope[scope]}, nil
}

func (f *scopedMorosidadDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakeMorosidadRow{err: pgx.ErrNoRows}
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
