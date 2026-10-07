package cuenta_corriente

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestListEntriesByUnidadReturnsDTOs(t *testing.T) {
	queries := db.New(fakeCuentaCorrienteDBTX{entries: []db.AccountEntry{
		stubEntry("cargo", 1500, 0, "cargo 1"),
		stubEntry("reversa", 0, 1500, "reversa 1"),
	}})

	items, err := ListEntriesByUnidad(context.Background(), queries, stubUUIDString("unit-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(items))
	}
	if items[0].Tipo != "cargo" || items[0].DebitCents != 1500 {
		t.Fatalf("first entry unexpected: %+v", items[0])
	}
	if items[1].Tipo != "reversa" || items[1].CreditCents != 1500 {
		t.Fatalf("second entry unexpected: %+v", items[1])
	}
}

func TestListEntriesByUnidadForwardsScopeAndReturnsOnlyScopedRows(t *testing.T) {
	const (
		scopeA = "00000000-0000-0000-0000-000000000002" // unit-1
		scopeB = "00000000-0000-0000-0000-000000000099" // unit-9
	)
	entryA := stubEntry("cargo", 1500, 0, "cargo scope A")
	entryB := stubEntry("cargo", 900, 0, "cargo scope B")
	entryB.UnidadID = stubUUID("unit-9")

	fake := &scopedCuentaCorrienteDBTX{entriesByScope: map[string][]db.AccountEntry{
		scopeA: {entryA},
		scopeB: {entryB},
	}}
	queries := db.New(fake)

	itemsA, err := ListEntriesByUnidad(context.Background(), queries, scopeA)
	if err != nil {
		t.Fatalf("unexpected error for scope A: %v", err)
	}
	if len(fake.calls) != 1 || fake.calls[0] != scopeA {
		t.Fatalf("expected scope A forwarded to the query, got calls %v", fake.calls)
	}
	if len(itemsA) != 1 || itemsA[0].UnidadID != scopeA || itemsA[0].DebitCents != 1500 {
		t.Fatalf("scope A expected only its own rows, got %+v", itemsA)
	}

	itemsB, err := ListEntriesByUnidad(context.Background(), queries, scopeB)
	if err != nil {
		t.Fatalf("unexpected error for scope B: %v", err)
	}
	if len(fake.calls) != 2 || fake.calls[1] != scopeB {
		t.Fatalf("expected scope B forwarded to the query, got calls %v", fake.calls)
	}
	if len(itemsB) != 1 || itemsB[0].UnidadID != scopeB || itemsB[0].DebitCents != 900 {
		t.Fatalf("scope B expected only its own rows, got %+v", itemsB)
	}

	for _, item := range itemsA {
		if item.UnidadID == scopeB {
			t.Fatalf("scope B row leaked into scope A result: %+v", item)
		}
	}
	for _, item := range itemsB {
		if item.UnidadID == scopeA {
			t.Fatalf("scope A row leaked into scope B result: %+v", item)
		}
	}
}

func TestListEntriesByUnidadInvalidScopeDoesNotQuery(t *testing.T) {
	fake := &scopedCuentaCorrienteDBTX{entriesByScope: map[string][]db.AccountEntry{}}
	queries := db.New(fake)

	_, err := ListEntriesByUnidad(context.Background(), queries, "no-es-uuid")
	if !errors.Is(err, ErrCargoInvalido) {
		t.Fatalf("expected ErrCargoInvalido, got %v", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("expected no query without a valid scope, got calls %v", fake.calls)
	}
}

type scopedCuentaCorrienteDBTX struct {
	entriesByScope map[string][]db.AccountEntry
	calls          []string
}

func (f *scopedCuentaCorrienteDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *scopedCuentaCorrienteDBTX) Query(_ context.Context, _ string, args ...interface{}) (pgx.Rows, error) {
	scope := scopedQueryKey(args)
	f.calls = append(f.calls, scope)
	entries := f.entriesByScope[scope]
	rows := make([][]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []any{e.TenantID, e.UnidadID, e.ID, e.Tipo, e.FechaEfectiva, e.DebitCents, e.CreditCents, e.Currency, e.Referencia, e.ReversaDeID, e.ChargeID, e.CreatedAt})
	}
	return &fakeRows{rows: rows}, nil
}

func (f *scopedCuentaCorrienteDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakeRow{err: pgx.ErrNoRows}
}

func scopedQueryKey(args []any) string {
	if len(args) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", args[0])
}

type fakeCuentaCorrienteDBTX struct {
	entries []db.AccountEntry
}

func (f fakeCuentaCorrienteDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f fakeCuentaCorrienteDBTX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	rows := make([][]any, 0, len(f.entries))
	for _, e := range f.entries {
		rows = append(rows, []any{e.TenantID, e.UnidadID, e.ID, e.Tipo, e.FechaEfectiva, e.DebitCents, e.CreditCents, e.Currency, e.Referencia, e.ReversaDeID, e.ChargeID, e.CreatedAt})
	}
	return &fakeRows{rows: rows}, nil
}

func (f fakeCuentaCorrienteDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakeRow{err: pgx.ErrNoRows}
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

func stubEntry(tipo string, debit, credit int64, ref string) db.AccountEntry {
	return db.AccountEntry{
		TenantID:      stubUUID("tenant-1"),
		UnidadID:      stubUUID("unit-1"),
		ID:            stubUUID(ref),
		Tipo:          tipo,
		FechaEfectiva: stubDate("2026-09-10"),
		DebitCents:    debit,
		CreditCents:   credit,
		Currency:      "ARS",
		Referencia:    stubText(ref),
		CreatedAt:     stubTime(),
	}
}

func stubUUIDString(seed string) string { return stubUUID(seed).String() }

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
	case "cargo 1":
		return "00000000-0000-0000-0000-000000000003"
	case "reversa 1":
		return "00000000-0000-0000-0000-000000000004"
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
