package expensas

import (
	"errors"
	"math"
	"reflect"
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

// Invariante de dinero (criterio de demo del roadmap): la suma por UF
// coincide con el total, concepto por concepto y en el agregado.
func TestDistribucionSumaIgualTotal(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100},
		{ConceptoID: "c2", Regla: "coeficiente", ImporteCents: 77777},
		{ConceptoID: "c3", Regla: "coeficiente", ImporteCents: 1},
		{ConceptoID: "c4", Regla: "coeficiente", ImporteCents: 0},
	}
	ufs := []UFCoef{
		{UnidadID: "u3", Codigo: "C", Coeficiente: 0.33},
		{UnidadID: "u1", Codigo: "A", Coeficiente: 0.33},
		{UnidadID: "u2", Codigo: "B", Coeficiente: 0.34},
		{UnidadID: "u4", Codigo: "D", Coeficiente: 0},
	}

	result, err := Distribuir(conceptos, ufs)
	if err != nil {
		t.Fatal(err)
	}

	var esperado int64
	for _, c := range conceptos {
		esperado += c.ImporteCents
	}
	if result.TotalGastosCents != esperado {
		t.Errorf("total_gastos = %d, want %d", result.TotalGastosCents, esperado)
	}

	var sumaUnidades int64
	for _, u := range result.Unidades {
		var sumaItems int64
		for _, it := range u.Items {
			sumaItems += it.ImporteCents
		}
		if sumaItems != u.TotalCents {
			t.Errorf("unidad %s: suma de items %d != total de la unidad %d", u.Codigo, sumaItems, u.TotalCents)
		}
		sumaUnidades += u.TotalCents
	}
	if sumaUnidades != esperado {
		t.Errorf("suma por UF = %d, want %d (total)", sumaUnidades, esperado)
	}
	if result.TotalDistribuidoCents != esperado {
		t.Errorf("total_distribuido = %d, want %d", result.TotalDistribuidoCents, esperado)
	}
	if result.DiferenciaCents != 0 {
		t.Errorf("diferencia = %d, want 0", result.DiferenciaCents)
	}
}

func TestDistribuirVacio(t *testing.T) {
	_, err := Distribuir(nil, nil)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

// --- Determinismo: tie-break por código (ADR-0006) fijado por tests ---

// La salida debe quedar ordenada por código UF, sin importar el orden
// de la entrada: la entrada va B primero y A debe quedar en Unidades[0].
func TestDistribuirTieBreakOrdenaPorCodigo(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100},
	}
	ufs := []UFCoef{
		{UnidadID: "u2", Codigo: "B", Coeficiente: 0.5},
		{UnidadID: "u1", Codigo: "A", Coeficiente: 0.5},
	}
	result, err := Distribuir(conceptos, ufs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unidades) != 2 {
		t.Fatalf("unidades = %d, want 2", len(result.Unidades))
	}
	if result.Unidades[0].Codigo != "A" {
		t.Errorf("Unidades[0].Codigo = %q, want %q (orden por código, no por entrada)", result.Unidades[0].Codigo, "A")
	}
	if result.Unidades[0].UnidadID != "u1" {
		t.Errorf("Unidades[0].UnidadID = %q, want %q", result.Unidades[0].UnidadID, "u1")
	}
	if result.Unidades[0].TotalCents != 50 || result.Unidades[1].TotalCents != 50 {
		t.Errorf("reparto = %d/%d, want 50/50", result.Unidades[0].TotalCents, result.Unidades[1].TotalCents)
	}
}

// El residuo de redondeo (+1 cent) lo recibe la UF de código más bajo
// entre las de coeficiente > 0, con montos exactos por UF.
func TestDistribuirResiduoAUnoMasBajo(t *testing.T) {
	// 100 sobre tres coeficientes iguales: 33.33 c/u => residuo 1.
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100},
	}

	t.Run("sin coeficientes ceros", func(t *testing.T) {
		ufs := []UFCoef{
			{UnidadID: "u3", Codigo: "C", Coeficiente: 1},
			{UnidadID: "u1", Codigo: "A", Coeficiente: 1},
			{UnidadID: "u2", Codigo: "B", Coeficiente: 1},
		}
		result, err := Distribuir(conceptos, ufs)
		if err != nil {
			t.Fatal(err)
		}
		esperado := map[string]int64{"A": 34, "B": 33, "C": 33}
		for _, u := range result.Unidades {
			if want := esperado[u.Codigo]; u.TotalCents != want {
				t.Errorf("unidad %s = %d, want %d", u.Codigo, u.TotalCents, want)
			}
		}
		if result.TotalDistribuidoCents != 100 {
			t.Errorf("total_distribuido = %d, want 100", result.TotalDistribuidoCents)
		}
	})

	t.Run("residuo salta la UF con coeficiente cero", func(t *testing.T) {
		ufs := []UFCoef{
			{UnidadID: "u1", Codigo: "A", Coeficiente: 0},
			{UnidadID: "u2", Codigo: "B", Coeficiente: 1},
			{UnidadID: "u3", Codigo: "C", Coeficiente: 1},
			{UnidadID: "u4", Codigo: "D", Coeficiente: 1},
		}
		result, err := Distribuir(conceptos, ufs)
		if err != nil {
			t.Fatal(err)
		}
		esperado := map[string]int64{"A": 0, "B": 34, "C": 33, "D": 33}
		for _, u := range result.Unidades {
			if want := esperado[u.Codigo]; u.TotalCents != want {
				t.Errorf("unidad %s = %d, want %d", u.Codigo, u.TotalCents, want)
			}
		}
		if result.TotalDistribuidoCents != 100 {
			t.Errorf("total_distribuido = %d, want 100", result.TotalDistribuidoCents)
		}
	})
}

// Mismos conceptos con la entrada de UFs en orden distinto: el resultado
// debe ser idéntico (DeepEqual) Y coincidir con el reparto concreto esperado,
// para que una regresión del algoritmo falle aunque ambas corridas fallen igual.
func TestDistribuirIndependienteDelOrdenDeEntrada(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100},
		{ConceptoID: "c2", Regla: "coeficiente", ImporteCents: 10},
	}
	base := []UFCoef{
		{UnidadID: "u1", Codigo: "A", Coeficiente: 1},
		{UnidadID: "u2", Codigo: "B", Coeficiente: 1},
		{UnidadID: "u3", Codigo: "C", Coeficiente: 1},
	}
	ascendente := make([]UFCoef, len(base))
	copy(ascendente, base)
	descendente := make([]UFCoef, len(base))
	for i, uf := range base {
		descendente[len(base)-1-i] = uf
	}

	r1, err := Distribuir(conceptos, ascendente)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Distribuir(conceptos, descendente)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r1, r2) {
		t.Errorf("el resultado depende del orden de la entrada:\n%+v\nvs\n%+v", r1, r2)
	}

	// Reparto concreto pinneado: c1 = 100 => 34/33/33 (residuo a A),
	// c2 = 10 => 4/3/3 (residuo a A). Totales: A=38, B=36, C=36.
	esperado := map[string]int64{"A": 38, "B": 36, "C": 36}
	if len(r1.Unidades) != 3 {
		t.Fatalf("unidades = %d, want 3", len(r1.Unidades))
	}
	for i, codigo := range []string{"A", "B", "C"} {
		if r1.Unidades[i].Codigo != codigo {
			t.Errorf("Unidades[%d].Codigo = %q, want %q", i, r1.Unidades[i].Codigo, codigo)
		}
		if got := r1.Unidades[i].TotalCents; got != esperado[codigo] {
			t.Errorf("unidad %s = %d, want %d", codigo, got, esperado[codigo])
		}
	}
	if r1.TotalDistribuidoCents != 110 {
		t.Errorf("total_distribuido = %d, want 110", r1.TotalDistribuidoCents)
	}
	if r1.DiferenciaCents != 0 {
		t.Errorf("diferencia = %d, want 0", r1.DiferenciaCents)
	}
}

// --- Rutas de error con sentinela exacta ---

func TestDistribuirConceptosSinUFs(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100},
	}
	_, err := Distribuir(conceptos, nil)
	if !errors.Is(err, ErrCalculoSinUFs) {
		t.Errorf("err = %v, want ErrCalculoSinUFs", err)
	}
}

func TestDistribuirTodosLosCoeficientesCero(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 100},
	}
	ufs := []UFCoef{
		{UnidadID: "u1", Codigo: "A", Coeficiente: 0},
		{UnidadID: "u2", Codigo: "B", Coeficiente: 0},
	}
	_, err := Distribuir(conceptos, ufs)
	if !errors.Is(err, ErrCalculoSinCoefCero) {
		t.Errorf("err = %v, want ErrCalculoSinCoefCero", err)
	}
}

func TestDistribuirUnicaUF(t *testing.T) {
	conceptos := []ConceptoCalculo{
		{ConceptoID: "c1", Regla: "coeficiente", ImporteCents: 123456},
	}
	ufs := []UFCoef{
		{UnidadID: "u1", Codigo: "A", Coeficiente: 1},
	}
	result, err := Distribuir(conceptos, ufs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unidades) != 1 {
		t.Fatalf("unidades = %d, want 1", len(result.Unidades))
	}
	if result.Unidades[0].TotalCents != 123456 {
		t.Errorf("unidad A = %d, want 123456", result.Unidades[0].TotalCents)
	}
	if result.TotalDistribuidoCents != 123456 {
		t.Errorf("total_distribuido = %d, want 123456", result.TotalDistribuidoCents)
	}
	if result.UnidadesAlcanzadas != 1 {
		t.Errorf("unidades_alcanzadas = %d, want 1", result.UnidadesAlcanzadas)
	}
}
