package comunicaciones

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestPublicarComunicadoEncolaOutbox(t *testing.T) {
	fake := &fakeComunicacionesDBTX{comunicado: stubComunicado("com-1", "borrador")}
	q := db.New(fake)

	res, err := PublicarComunicado(context.Background(), q, validUUID("com-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Estado != "publicado" {
		t.Fatalf("estado unexpected: %s", res.Estado)
	}
	if fake.outboxEventType != "comunicado.publicado" {
		t.Fatalf("outbox event unexpected: %s", fake.outboxEventType)
	}
	if !strings.Contains(string(fake.outboxPayload), validUUID("com-1")) {
		t.Fatalf("payload missing comunicado id: %s", string(fake.outboxPayload))
	}
}

type fakeComunicacionesDBTX struct {
	comunicado      db.Comunicado
	outboxEventType string
	outboxPayload   []byte
}

func (f *fakeComunicacionesDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *fakeComunicacionesDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

func (f *fakeComunicacionesDBTX) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	switch {
	case strings.Contains(query, "FROM comunicados"):
		return fakeComunicacionesRow{values: comunicadoValues(f.comunicado)}
	case strings.Contains(query, "UPDATE comunicados"):
		return fakeComunicacionesRow{values: comunicadoValues(stubComunicado("com-1", "publicado"))}
	case strings.Contains(query, "INSERT INTO outbox_events"):
		f.outboxEventType = args[1].(string)
		f.outboxPayload = args[2].([]byte)
		return fakeComunicacionesRow{values: outboxValues()}
	default:
		return fakeComunicacionesRow{err: pgx.ErrNoRows}
	}
}

type fakeComunicacionesRow struct {
	values []any
	err    error
}

func (r fakeComunicacionesRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(r.values[i]))
	}
	return nil
}

func comunicadoValues(c db.Comunicado) []any {
	return []any{c.TenantID, c.ConsorcioID, c.ID, c.Titulo, c.Cuerpo, c.Destinatarios, c.Estado, c.PublicadoAt, c.CreatedBy, c.CreatedAt, c.UpdatedAt}
}

func outboxValues() []any {
	var ts pgtype.Timestamptz
	_ = ts.Scan(time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	return []any{stubUUID("tenant-1"), stubUUID("outbox-1"), stubUUID("com-1"), "comunicado.publicado", json.RawMessage(`{}`), "pendiente", int32(0), ts, ts}
}

func stubComunicado(id, estado string) db.Comunicado {
	var ts pgtype.Timestamptz
	_ = ts.Scan(time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	return db.Comunicado{TenantID: stubUUID("tenant-1"), ConsorcioID: stubUUID("consorcio-1"), ID: stubUUID(id), Titulo: "Aviso", Cuerpo: "Cuerpo", Destinatarios: "todos", Estado: estado, CreatedBy: stubUUID("user-1"), CreatedAt: ts, UpdatedAt: ts}
}

func stubUUID(seed string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(validUUID(seed))
	return u
}

func validUUID(seed string) string {
	switch seed {
	case "tenant-1":
		return "00000000-0000-0000-0000-000000000121"
	case "consorcio-1":
		return "00000000-0000-0000-0000-000000000122"
	case "user-1":
		return "00000000-0000-0000-0000-000000000123"
	case "com-1":
		return "00000000-0000-0000-0000-000000000124"
	case "outbox-1":
		return "00000000-0000-0000-0000-000000000125"
	default:
		return "00000000-0000-0000-0000-000000000199"
	}
}
