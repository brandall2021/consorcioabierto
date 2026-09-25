package reclamos

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

func TestTransicionarReclamoCambiaEstadoYRegistraHistorial(t *testing.T) {
	fake := &fakeReclamosDBTX{reclamo: stubReclamo("rec-1", "abierto")}
	q := db.New(fake)

	res, err := TransicionarReclamo(context.Background(), q, validReclamoUUID("rec-1"), validReclamoUUID("user-1"), TransitionInput{Accion: "en_progreso", Motivo: strPtr("atendiendo"), ResponsableID: strPtr(validReclamoUUID("user-2"))})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Estado != "en_progreso" {
		t.Fatalf("estado unexpected: %s", res.Estado)
	}
	if fake.ultimaAccion != "en_progreso" {
		t.Fatalf("transition action unexpected: %s", fake.ultimaAccion)
	}
	if fake.ultimoMotivo != "atendiendo" {
		t.Fatalf("transition motive unexpected: %s", fake.ultimoMotivo)
	}
}

type fakeReclamosDBTX struct {
	reclamo      db.Reclamo
	ultimaAccion string
	ultimoMotivo string
}

func (f *fakeReclamosDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *fakeReclamosDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

func (f *fakeReclamosDBTX) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	switch {
	case strings.Contains(query, "FROM reclamos"):
		return fakeReclamosRow{values: reclamoValues(f.reclamo)}
	case strings.Contains(query, "UPDATE reclamos"):
		updated := f.reclamo
		if state, ok := args[0].(pgtype.Text); ok && state.Valid {
			updated.Estado = state.String
		}
		if resp, ok := args[1].(pgtype.UUID); ok && resp != (pgtype.UUID{}) {
			updated.ResponsableID = resp
		}
		return fakeReclamosRow{values: reclamoValues(updated)}
	case strings.Contains(query, "INSERT INTO reclamo_transiciones"):
		f.ultimaAccion = args[1].(string)
		if args[2] != nil {
			f.ultimoMotivo = args[2].(pgtype.Text).String
		}
		return fakeReclamosRow{values: transicionValues()}
	default:
		return fakeReclamosRow{err: pgx.ErrNoRows}
	}
}

type fakeReclamosRow struct {
	values []any
	err    error
}

func (r fakeReclamosRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(r.values[i]))
	}
	return nil
}

func reclamoValues(c db.Reclamo) []any {
	return []any{c.TenantID, c.ConsorcioID, c.UnidadID, c.ID, c.Categoria, c.Texto, c.Estado, c.ResponsableID, c.CreatedBy, c.CreatedAt, c.UpdatedAt}
}

func transicionValues() []any {
	var ts pgtype.Timestamptz
	_ = ts.Scan(time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	return []any{stubReclamoUUID("tenant-1"), stubReclamoUUID("rec-1"), stubReclamoUUID("trans-1"), "en_progreso", pgtype.Text{String: "atendiendo", Valid: true}, stubReclamoUUID("user-2"), stubReclamoUUID("user-1"), ts}
}

func stubReclamo(id, estado string) db.Reclamo {
	var ts pgtype.Timestamptz
	_ = ts.Scan(time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	return db.Reclamo{TenantID: stubReclamoUUID("tenant-1"), ConsorcioID: stubReclamoUUID("consorcio-1"), UnidadID: stubReclamoUUID("unidad-1"), ID: stubReclamoUUID(id), Categoria: "agua", Texto: "Fuga", Estado: estado, CreatedBy: stubReclamoUUID("user-1"), CreatedAt: ts, UpdatedAt: ts}
}

func stubReclamoUUID(seed string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(validReclamoUUID(seed))
	return u
}

func validReclamoUUID(seed string) string {
	switch seed {
	case "tenant-1":
		return "00000000-0000-0000-0000-000000000221"
	case "consorcio-1":
		return "00000000-0000-0000-0000-000000000222"
	case "unidad-1":
		return "00000000-0000-0000-0000-000000000223"
	case "user-1":
		return "00000000-0000-0000-0000-000000000224"
	case "user-2":
		return "00000000-0000-0000-0000-000000000225"
	case "rec-1":
		return "00000000-0000-0000-0000-000000000226"
	case "trans-1":
		return "00000000-0000-0000-0000-000000000227"
	default:
		return "00000000-0000-0000-0000-000000000299"
	}
}

func strPtr(s string) *string { return &s }
