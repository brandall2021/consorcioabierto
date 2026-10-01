package portal

import (
	"context"
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestBuildHomeSummarizesOverdueDebtAndRecentReceipts(t *testing.T) {
	fake := &homeQueryFake{}
	res, err := BuildHome(context.Background(), fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalSaldoVencidoCents != 15000 {
		t.Fatalf("expected total overdue balance 15000, got %d", res.TotalSaldoVencidoCents)
	}
	if len(res.Consorcios) != 1 || res.Consorcios[0].SaldoVencidoCents != 15000 {
		t.Fatalf("unexpected consorcios: %+v", res.Consorcios)
	}
	if len(res.RecibosRecientes) != 1 || res.RecibosRecientes[0].ConsorcioNombre != "Consorcio Norte" {
		t.Fatalf("unexpected receipts: %+v", res.RecibosRecientes)
	}
	if len(res.ComunicadosRecientes) != 0 || len(res.ReclamosRecientes) != 0 {
		t.Fatalf("expected empty recent comunicados/reclamos, got %+v %+v", res.ComunicadosRecientes, res.ReclamosRecientes)
	}
}

type homeQueryFake struct{}

func (f *homeQueryFake) ListConsorcios(context.Context, db.ListConsorciosParams) ([]db.Consorcio, error) {
	return []db.Consorcio{{ID: mustUUID("11111111-1111-4111-8111-111111111111"), Nombre: "Consorcio Norte", Estado: "activo", CreatedAt: mustTime()}}, nil
}

func (f *homeQueryFake) ListUnidades(context.Context, db.ListUnidadesParams) ([]db.Unidade, error) {
	return []db.Unidade{{ID: mustUUID("22222222-2222-4222-8222-222222222222"), ConsorcioID: mustUUID("11111111-1111-4111-8111-111111111111"), Codigo: "1A", Tipo: "departamento", Estado: "activo", CreatedAt: mustTime()}}, nil
}

func (f *homeQueryFake) ListUnidadesForCurrentUser(context.Context) ([]db.Unidade, error) {
	return []db.Unidade{{ID: mustUUID("22222222-2222-4222-8222-222222222222"), ConsorcioID: mustUUID("11111111-1111-4111-8111-111111111111"), Codigo: "1A", Tipo: "departamento", Estado: "activo", CreatedAt: mustTime()}}, nil
}

func (f *homeQueryFake) ListOpenChargesByUnidad(context.Context, db.ListOpenChargesByUnidadParams) ([]db.Charge, error) {
	return []db.Charge{{ID: mustUUID("33333333-3333-4333-8333-333333333333"), UnidadID: mustUUID("22222222-2222-4222-8222-222222222222"), Concepto: "Expensa julio", DueDate: mustDate("2026-09-01"), TotalCents: 15000, SaldoCents: 15000, CreatedAt: mustTime()}}, nil
}

func (f *homeQueryFake) ListComunicados(context.Context, db.ListComunicadosParams) ([]db.Comunicado, error) {
	return nil, nil
}

func (f *homeQueryFake) ListReclamos(context.Context, db.ListReclamosParams) ([]db.Reclamo, error) {
	return nil, nil
}

func (f *homeQueryFake) ListCobranzas(context.Context, pgtype.UUID) ([]db.Payment, error) {
	return []db.Payment{{ID: mustUUID("44444444-4444-4444-8444-444444444444"), UnidadID: mustUUID("22222222-2222-4222-8222-222222222222"), Fecha: mustDate("2026-09-10"), Canal: "efectivo", ImporteCents: 5000, Estado: "acreditado", CreatedAt: mustTime(), Referencia: pgtype.Text{String: "REC-1", Valid: true}}}, nil
}

func (f *homeQueryFake) ListPaymentAllocationsByPayment(context.Context, pgtype.UUID) ([]db.PaymentAllocation, error) {
	return []db.PaymentAllocation{{AmountCents: 5000}}, nil
}

var _ Queryer = (*homeQueryFake)(nil)

var _ = time.Time{}

func mustUUID(s string) pgtype.UUID {
	var out pgtype.UUID
	_ = out.Scan(s)
	return out
}

func mustDate(s string) pgtype.Date {
	var out pgtype.Date
	_ = out.Scan(s)
	return out
}

func mustTime() pgtype.Timestamptz {
	var out pgtype.Timestamptz
	_ = out.Scan(time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC))
	return out
}

// El portal del consorcista solo debe exponer sus propias UF. Este fake devuelve
// un consorcio con dos unidades, pero ListUnidadesForCurrentUser solo devuelve
// la del usuario: deuda, recibos y reclamos de la otra unidad deben quedar fuera.
type aislamientoQueryFake struct{}

func (f *aislamientoQueryFake) ListConsorcios(context.Context, db.ListConsorciosParams) ([]db.Consorcio, error) {
	return []db.Consorcio{{ID: mustUUID("11111111-1111-4111-8111-111111111111"), Nombre: "Consorcio Norte", Estado: "activo", CreatedAt: mustTime()}}, nil
}

func (f *aislamientoQueryFake) ListUnidades(context.Context, db.ListUnidadesParams) ([]db.Unidade, error) {
	return []db.Unidade{
		{ID: mustUUID("aaaa1111-1111-4111-8111-111111111111"), ConsorcioID: mustUUID("11111111-1111-4111-8111-111111111111"), Codigo: "1A", Estado: "activo", CreatedAt: mustTime()},
		{ID: mustUUID("bbbb2222-2222-4222-8222-222222222222"), ConsorcioID: mustUUID("11111111-1111-4111-8111-111111111111"), Codigo: "1B", Estado: "activo", CreatedAt: mustTime()},
	}, nil
}

func (f *aislamientoQueryFake) ListUnidadesForCurrentUser(context.Context) ([]db.Unidade, error) {
	return []db.Unidade{{ID: mustUUID("aaaa1111-1111-4111-8111-111111111111"), ConsorcioID: mustUUID("11111111-1111-4111-8111-111111111111"), Codigo: "1A", Estado: "activo", CreatedAt: mustTime()}}, nil
}

func (f *aislamientoQueryFake) ListOpenChargesByUnidad(_ context.Context, arg db.ListOpenChargesByUnidadParams) ([]db.Charge, error) {
	if arg.UnidadID == mustUUID("bbbb2222-2222-4222-8222-222222222222") {
		return []db.Charge{{UnidadID: arg.UnidadID, DueDate: mustDate("2026-01-01"), SaldoCents: 99000, CreatedAt: mustTime()}}, nil
	}
	return []db.Charge{{UnidadID: arg.UnidadID, DueDate: mustDate("2026-09-01"), SaldoCents: 15000, CreatedAt: mustTime()}}, nil
}

func (f *aislamientoQueryFake) ListComunicados(context.Context, db.ListComunicadosParams) ([]db.Comunicado, error) {
	return nil, nil
}

// ListReclamos devuelve los reclamos de todo el consorcio, como hace la query
// real. BuildHome debe descartar los que no pertenecen a una UF del usuario.
func (f *aislamientoQueryFake) ListReclamos(context.Context, db.ListReclamosParams) ([]db.Reclamo, error) {
	return []db.Reclamo{
		{ID: mustUUID("cccc3333-3333-4333-8333-333333333333"), UnidadID: mustUUID("aaaa1111-1111-4111-8111-111111111111"), Categoria: "plomería", Estado: "abierto", Texto: "Fuga en cocina", CreatedAt: mustTime()},
		{ID: mustUUID("dddd4444-4444-4444-8444-444444444444"), UnidadID: mustUUID("bbbb2222-2222-4222-8222-222222222222"), Categoria: "ascensor", Estado: "abierto", Texto: "Ascensor del vecino", CreatedAt: mustTime()},
	}, nil
}

func (f *aislamientoQueryFake) ListCobranzas(context.Context, pgtype.UUID) ([]db.Payment, error) {
	return []db.Payment{
		{ID: mustUUID("eeee5555-5555-4555-8555-555555555555"), UnidadID: mustUUID("aaaa1111-1111-4111-8111-111111111111"), Fecha: mustDate("2026-09-10"), ImporteCents: 5000, Estado: "acreditado", CreatedAt: mustTime()},
		{ID: mustUUID("ffff6666-6666-4666-8666-666666666666"), UnidadID: mustUUID("bbbb2222-2222-4222-8222-222222222222"), Fecha: mustDate("2026-09-11"), ImporteCents: 77000, Estado: "acreditado", CreatedAt: mustTime()},
	}, nil
}

func (f *aislamientoQueryFake) ListPaymentAllocationsByPayment(context.Context, pgtype.UUID) ([]db.PaymentAllocation, error) {
	return []db.PaymentAllocation{{AmountCents: 5000}}, nil
}

var _ Queryer = (*aislamientoQueryFake)(nil)

func TestBuildHomeNoExponeUnidadesAjenas(t *testing.T) {
	res, err := BuildHome(context.Background(), &aislamientoQueryFake{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalSaldoVencidoCents != 15000 {
		t.Fatalf("deuda de unidad ajena filtrada al total: %d", res.TotalSaldoVencidoCents)
	}
	if len(res.Consorcios) != 1 || res.Consorcios[0].SaldoVencidoCents != 15000 {
		t.Fatalf("resumen de consorcio incluye deuda ajena: %+v", res.Consorcios)
	}
	if len(res.RecibosRecientes) != 1 {
		t.Fatalf("esperaba 1 recibo propio, obtuve %d: %+v", len(res.RecibosRecientes), res.RecibosRecientes)
	}
	if res.RecibosRecientes[0].CobranzaID != "eeee5555-5555-4555-8555-555555555555" {
		t.Fatalf("se expuso el recibo de otra unidad: %+v", res.RecibosRecientes[0])
	}
	if len(res.ReclamosRecientes) != 1 {
		t.Fatalf("esperaba 1 reclamo propio, obtuve %d: %+v", len(res.ReclamosRecientes), res.ReclamosRecientes)
	}
	if res.ReclamosRecientes[0].UnidadID != "aaaa1111-1111-4111-8111-111111111111" {
		t.Fatalf("se expuso el reclamo de otra unidad: %+v", res.ReclamosRecientes[0])
	}
}

func TestBuildHomeSinVinculoNoExponeNadaDelTenant(t *testing.T) {
	res, err := BuildHome(context.Background(), &sinVinculoQueryFake{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalSaldoVencidoCents != 0 || len(res.Consorcios) != 0 ||
		len(res.RecibosRecientes) != 0 || len(res.ReclamosRecientes) != 0 {
		t.Fatalf("usuario sin vinculo recibio datos del tenant: %+v", res)
	}
}

type sinVinculoQueryFake struct{ aislamientoQueryFake }

func (f *sinVinculoQueryFake) ListUnidadesForCurrentUser(context.Context) ([]db.Unidade, error) {
	return nil, nil
}
