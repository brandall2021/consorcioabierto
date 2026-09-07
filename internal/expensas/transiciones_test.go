package expensas

import (
	"testing"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

func uid(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		t.Fatalf("scan uuid %q: %v", s, err)
	}
	return u
}

func gasto(id, conceptoID string, importe int64) db.Gasto {
	gi, ci := pgtype.UUID{}, pgtype.UUID{}
	_ = gi.Scan(id)
	_ = ci.Scan(conceptoID)
	return db.Gasto{ID: gi, ConceptoID: ci, ImporteCents: importe}
}

func TestAgruparConceptosSumaPorConcepto(t *testing.T) {
	gastos := []db.Gasto{
		gasto("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-00000000000a", 1000),
		gasto("00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-00000000000a", 2000),
		gasto("00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-00000000000b", 500),
	}
	got := agruparConceptos(gastos)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ConceptoID != "00000000-0000-0000-0000-00000000000a" {
		t.Errorf("concepto[0] = %q", got[0].ConceptoID)
	}
	if got[0].ImporteCents != 3000 {
		t.Errorf("concepto[0] importe = %d, want 3000", got[0].ImporteCents)
	}
	if got[0].Regla != "coeficiente" {
		t.Errorf("regla[0] = %q, want coeficiente", got[0].Regla)
	}
	if got[1].ImporteCents != 500 {
		t.Errorf("concepto[1] importe = %d, want 500", got[1].ImporteCents)
	}
}

func TestAgruparConceptosPreservaOrden(t *testing.T) {
	gastos := []db.Gasto{
		gasto("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-00000000000b", 100),
		gasto("00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-00000000000a", 200),
		gasto("00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-00000000000b", 300),
	}
	got := agruparConceptos(gastos)
	if got[0].ConceptoID != "00000000-0000-0000-0000-00000000000b" {
		t.Errorf("primer concepto = %q, want el b primero (orden de aparición)", got[0].ConceptoID)
	}
	if got[0].ImporteCents != 400 {
		t.Errorf("b total = %d, want 400", got[0].ImporteCents)
	}
}

func TestAgruparConceptosVacio(t *testing.T) {
	got := agruparConceptos(nil)
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestToUnidadCobro(t *testing.T) {
	rows := []db.GetLiquidacionUnidadesRow{
		{UnidadID: uid(t, "00000000-0000-0000-0000-000000000011"), Codigo: "A1", TotalCents: 1000},
		{UnidadID: uid(t, "00000000-0000-0000-0000-000000000012"), Codigo: "A2", TotalCents: 0},
		{UnidadID: uid(t, "00000000-0000-0000-0000-000000000013"), Codigo: "A3", TotalCents: 250},
	}
	liqID := "00000000-0000-0000-0000-0000000000aa"
	got := toUnidadCobro(rows, "202609", liqID, "2026-10-05")

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (fila con total 0 se omite)", len(got))
	}
	if got[0].UnidadID != "00000000-0000-0000-0000-000000000011" {
		t.Errorf("unidad[0] = %q", got[0].UnidadID)
	}
	if got[0].DueDate != "2026-10-05" {
		t.Errorf("due_date[0] = %q, want vencimiento_1", got[0].DueDate)
	}
	if got[0].Concepto != "Expensa 202609" {
		t.Errorf("concepto[0] = %q, want 'Expensa 202609'", got[0].Concepto)
	}
	if got[0].TotalCents != 1000 {
		t.Errorf("total[0] = %d, want 1000", got[0].TotalCents)
	}
	if got[0].LiquidacionID == nil || *got[0].LiquidacionID != liqID {
		t.Errorf("liquidacion_id[0] = %v, want %q", got[0].LiquidacionID, liqID)
	}
	if got[1].UnidadID != "00000000-0000-0000-0000-000000000013" || got[1].TotalCents != 250 {
		t.Errorf("unidad[1] = %q/%d, want A3/250", got[1].UnidadID, got[1].TotalCents)
	}
}