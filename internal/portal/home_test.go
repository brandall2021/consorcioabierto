package portal

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

func TestBuildTenantHomeAggregatesDebtAndReceipts(t *testing.T) {
	queries := db.New(fakePortalDBTX{
		consorcios: []db.Consorcio{
			stubConsorcio("consorcio-1", "Consorcio A"),
			stubConsorcio("consorcio-2", "Consorcio B"),
		},
		morosidad: map[string][]db.ListMorosidadByConsorcioRow{
			stubUUIDString("consorcio-1"): {{UnidadID: stubUUID("unit-1"), UnidadCodigo: "UF 1", SaldoVencidoCents: 5000, CantidadCargos: 2, VencidoDesde: stubDate("2026-08-01")}},
			stubUUIDString("consorcio-2"): {{UnidadID: stubUUID("unit-2"), UnidadCodigo: "UF 9", SaldoVencidoCents: 2000, CantidadCargos: 1, VencidoDesde: stubDate("2026-08-15")}},
		},
		cobranzas: map[string][]db.Payment{
			stubUUIDString("consorcio-1"): {stubPayment("payment-1", "2026-09-10", 7000)},
			stubUUIDString("consorcio-2"): {stubPayment("payment-2", "2026-09-11", 3000)},
		},
		comunicados: map[string][]db.Comunicado{
			stubUUIDString("consorcio-1"): {stubComunicado("com-1", "2026-09-12", "Reunión"), stubComunicado("com-2", "2026-09-11", "Corte de agua")},
		},
		reclamos: map[string][]db.Reclamo{
			stubUUIDString("consorcio-1"): {stubReclamo("rec-1", "abierto", "Fuga en cocina")},
		},
	})

	home, err := BuildTenantHome(context.Background(), queries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if home.TotalSaldoVencidoCents != 7000 {
		t.Fatalf("saldo total unexpected: %d", home.TotalSaldoVencidoCents)
	}
	if len(home.Consorcios) != 2 {
		t.Fatalf("consorcios unexpected: %d", len(home.Consorcios))
	}
	if len(home.RecibosRecientes) != 2 {
		t.Fatalf("recibos unexpected: %d", len(home.RecibosRecientes))
	}
	if home.RecibosRecientes[0].CobranzaID != stubUUIDString("payment-2") {
		t.Fatalf("expected latest receipt first, got %+v", home.RecibosRecientes[0])
	}
	if len(home.ComunicadosRecientes) != 2 {
		t.Fatalf("comunicados unexpected: %d", len(home.ComunicadosRecientes))
	}
	if home.ComunicadosRecientes[0].ID != stubUUIDString("com-1") {
		t.Fatalf("expected latest comunicado first, got %+v", home.ComunicadosRecientes[0])
	}
	if len(home.ReclamosRecientes) != 1 {
		t.Fatalf("reclamos unexpected: %d", len(home.ReclamosRecientes))
	}
}

type fakePortalDBTX struct {
	consorcios  []db.Consorcio
	morosidad   map[string][]db.ListMorosidadByConsorcioRow
	cobranzas   map[string][]db.Payment
	comunicados map[string][]db.Comunicado
	reclamos    map[string][]db.Reclamo
}

func (f fakePortalDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f fakePortalDBTX) Query(_ context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	switch {
	case strings.Contains(query, "FROM consorcios"):
		return &fakePortalRows{rows: consorcioRows(f.consorcios)}, nil
	case strings.Contains(query, "GROUP BY u.id, u.codigo"):
		consorcioID := args[0].(pgtype.UUID).String()
		return &fakePortalRows{rows: morosidadRows(f.morosidad[consorcioID])}, nil
	case strings.Contains(query, "FROM payments"):
		consorcioID := args[0].(pgtype.UUID).String()
		return &fakePortalRows{rows: paymentRows(f.cobranzas[consorcioID])}, nil
	case strings.Contains(query, "FROM comunicados"):
		consorcioID := args[0].(pgtype.UUID).String()
		return &fakePortalRows{rows: comunicadoRows(f.comunicados[consorcioID])}, nil
	case strings.Contains(query, "FROM reclamos"):
		consorcioID := args[0].(pgtype.UUID).String()
		return &fakePortalRows{rows: reclamoRows(f.reclamos[consorcioID])}, nil
	default:
		return nil, pgx.ErrNoRows
	}
}

func (f fakePortalDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakePortalRow{err: pgx.ErrNoRows}
}

type fakePortalRow struct{ err error }

func (r fakePortalRow) Scan(dest ...any) error { return r.err }

type fakePortalRows struct {
	rows [][]any
	idx  int
}

func (r *fakePortalRows) Close()                                       {}
func (r *fakePortalRows) Err() error                                   { return nil }
func (r *fakePortalRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakePortalRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakePortalRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakePortalRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}
func (r *fakePortalRows) Values() ([]any, error) { return r.rows[r.idx-1], nil }
func (r *fakePortalRows) RawValues() [][]byte    { return nil }
func (r *fakePortalRows) Conn() *pgx.Conn        { return nil }

func consorcioRows(rows []db.Consorcio) [][]any {
	out := make([][]any, 0, len(rows))
	for _, c := range rows {
		out = append(out, []any{c.TenantID, c.ID, c.Nombre, c.Cuit, c.Domicilio, c.Tipo, c.Estado, c.Config, c.CreatedAt, c.UpdatedAt})
	}
	return out
}

func morosidadRows(rows []db.ListMorosidadByConsorcioRow) [][]any {
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{r.UnidadID, r.UnidadCodigo, r.SaldoVencidoCents, r.CantidadCargos, r.VencidoDesde})
	}
	return out
}

func paymentRows(rows []db.Payment) [][]any {
	out := make([][]any, 0, len(rows))
	for _, p := range rows {
		out = append(out, []any{p.TenantID, p.UnidadID, p.ID, p.Fecha, p.Canal, p.ImporteCents, p.Referencia, p.Estado, p.IdemKey, p.MotivoRechazo, p.CreatedBy, p.CreatedAt, p.PspProvider, p.PspPreferenceID, p.PspCheckoutUrl})
	}
	return out
}

func comunicadoRows(rows []db.Comunicado) [][]any {
	out := make([][]any, 0, len(rows))
	for _, c := range rows {
		out = append(out, []any{c.TenantID, c.ConsorcioID, c.ID, c.Titulo, c.Cuerpo, c.Destinatarios, c.Estado, c.PublicadoAt, c.CreatedBy, c.CreatedAt, c.UpdatedAt})
	}
	return out
}

func reclamoRows(rows []db.Reclamo) [][]any {
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{r.TenantID, r.ConsorcioID, r.UnidadID, r.ID, r.Categoria, r.Texto, r.Estado, r.ResponsableID, r.CreatedBy, r.CreatedAt, r.UpdatedAt})
	}
	return out
}

func stubConsorcio(id, nombre string) db.Consorcio {
	return db.Consorcio{TenantID: stubUUID("tenant-1"), ID: stubUUID(id), Nombre: nombre, Tipo: "edificio", Estado: "activo", CreatedAt: stubTimestamptz()}
}

func stubPayment(id, fecha string, amount int64) db.Payment {
	return db.Payment{TenantID: stubUUID("tenant-1"), UnidadID: stubUUID("unit-1"), ID: stubUUID(id), Fecha: stubDate(fecha), Canal: "transferencia", ImporteCents: amount, Estado: "acreditado", CreatedAt: stubTimestamptz()}
}

func stubComunicado(id, fecha, titulo string) db.Comunicado {
	ts := stubTimestamptzAt(fecha)
	return db.Comunicado{TenantID: stubUUID("tenant-1"), ConsorcioID: stubUUID("consorcio-1"), ID: stubUUID(id), Titulo: titulo, Cuerpo: "Cuerpo", Destinatarios: "todos", Estado: "publicado", PublicadoAt: ts, CreatedBy: stubUUID("user-1"), CreatedAt: ts, UpdatedAt: ts}
}

func stubReclamo(id, estado, texto string) db.Reclamo {
	ts := stubTimestamptzAt("2026-09-10")
	return db.Reclamo{TenantID: stubUUID("tenant-1"), ConsorcioID: stubUUID("consorcio-1"), UnidadID: stubUUID("unit-1"), ID: stubUUID(id), Categoria: "agua", Texto: texto, Estado: estado, CreatedBy: stubUUID("user-1"), CreatedAt: ts, UpdatedAt: ts}
}

func stubUUID(seed string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(validPortalUUID(seed))
	return u
}

func stubUUIDString(seed string) string {
	return stubUUID(seed).String()
}

func validPortalUUID(seed string) string {
	switch seed {
	case "tenant-1":
		return "00000000-0000-0000-0000-000000000021"
	case "consorcio-1":
		return "00000000-0000-0000-0000-000000000022"
	case "consorcio-2":
		return "00000000-0000-0000-0000-000000000023"
	case "unit-1":
		return "00000000-0000-0000-0000-000000000024"
	case "unit-2":
		return "00000000-0000-0000-0000-000000000025"
	case "payment-1":
		return "00000000-0000-0000-0000-000000000026"
	case "payment-2":
		return "00000000-0000-0000-0000-000000000027"
	default:
		return "00000000-0000-0000-0000-000000000099"
	}
}

func stubDate(date string) pgtype.Date {
	var d pgtype.Date
	_ = d.Scan(date)
	return d
}

func stubTimestamptz() pgtype.Timestamptz {
	var ts pgtype.Timestamptz
	_ = ts.Scan(time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	return ts
}

func stubTimestamptzAt(date string) pgtype.Timestamptz {
	var ts pgtype.Timestamptz
	_ = ts.Scan(date + "T12:00:00Z")
	return ts
}
