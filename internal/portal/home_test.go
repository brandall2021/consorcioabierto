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
