package expensas

import (
	"math"
	"testing"
)

func TestDistribuirSimple(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100000},
	}
	ufs := []UFCoef{
		{UnidadID: "u1", Codigo: "A1", Coeficiente: 0.6},
		{UnidadID: "u2", Codigo: "A2", Coeficiente: 0.4},
	}
	result, err := Distribuir(conceptos, ufs)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalGastosCents != 100000 {
		t.Errorf("total_gastos = %d, want 100000", result.TotalGastosCents)
	}
	if result.TotalDistribuidoCents != 100000 {
		t.Errorf("total_distribuido = %d, want 100000", result.TotalDistribuidoCents)
	}
	if result.UnidadesAlcanzadas != 2 {
		t.Errorf("unidades_alcanzadas = %d, want 2", result.UnidadesAlcanzadas)
	}
	// Verificar que suma de unidades == total distribuido
	var suma int64
	for _, u := range result.Unidades {
		suma += u.TotalCents
	}
	if suma != result.TotalDistribuidoCents {
		t.Errorf("suma unidades %d != total_distribuido %d", suma, result.TotalDistribuidoCents)
	}
	// Verificar proporciones aproximadas
	u1 := result.Unidades[0]
	u2 := result.Unidades[1]
	ratio := float64(u1.TotalCents) / float64(u2.TotalCents)
	if math.Abs(ratio-1.5) > 0.01 {
		t.Errorf("ratio u1/u2 = %f, want ~1.5", ratio)
	}
}

func TestDistribuirDeterministico(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 123456},
	}
	ufs := []UFCoef{
		{UnidadID: "u3", Codigo: "C", Coeficiente: 0.33},
		{UnidadID: "u1", Codigo: "A", Coeficiente: 0.33},
		{UnidadID: "u2", Codigo: "B", Coeficiente: 0.34},
	}
	r1, _ := Distribuir(conceptos, ufs)
	r2, _ := Distribuir(conceptos, ufs)
	if r1.TotalDistribuidoCents != r2.TotalDistribuidoCents {
		t.Error("no determinista: mismos inputs, distintos totales")
	}
	for i := range r1.Unidades {
		if r1.Unidades[i].TotalCents != r2.Unidades[i].TotalCents {
			t.Errorf("unidad %d: %d != %d", i, r1.Unidades[i].TotalCents, r2.Unidades[i].TotalCents)
		}
	}
}

func TestDistribuirCoeficienteCero(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 50000},
	}
	ufs := []UFCoef{
		{UnidadID: "u1", Codigo: "A", Coeficiente: 0},
		{UnidadID: "u2", Codigo: "B", Coeficiente: 1.0},
	}
	result, err := Distribuir(conceptos, ufs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Advertencias) == 0 {
		t.Error("expected warning for zero coefficient")
	}
	if result.Unidades[0].TotalCents != 0 {
		t.Errorf("u1 (coef=0) should get 0, got %d", result.Unidades[0].TotalCents)
	}
	if result.Unidades[1].TotalCents != 50000 {
		t.Errorf("u2 should get 50000, got %d", result.Unidades[1].TotalCents)
	}
}

func TestDistribuirTieBreakCodigo(t *testing.T) {
	// Residuo de redondeo se asigna por código ascendente
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100},
	}
	ufs := []UFCoef{
		{UnidadID: "u2", Codigo: "B", Coeficiente: 0.5},
		{UnidadID: "u1", Codigo: "A", Coeficiente: 0.5},
	}
	result, _ := Distribuir(conceptos, ufs)
	// 100 * 0.5 = 50 each, no residuo
	if result.Unidades[0].TotalCents+result.Unidades[1].TotalCents != 100 {
		t.Error("total mismatch")
	}
	// Tie-break: A gets residuo first
	var suma int64
	for _, u := range result.Unidades {
		suma += u.TotalCents
	}
	if suma != 100 {
		t.Errorf("suma = %d, want 100", suma)
	}
}

func TestDistribuirVacio(t *testing.T) {
	_, err := Distribuir(nil, nil)
	if err == nil {
		t.Error("expected error for empty input")
	}
}
