package consorcios

import (
	"errors"
	"strings"
	"testing"
)

const csvHeader = "codigo,tipo,superficie,coeficiente,nombre,documento,email,telefono,vinculo,porcentaje\n"

func csvPlants(rows ...string) string { return csvHeader + strings.Join(rows, "\n") + "\n" }

func TestParsePlantillaUFSCSVValid(t *testing.T) {
	body := csvPlants(
		"1A,departamento,45.5,1.25,Juan Perez,DNI 30111222,juan@correo.com,1155551111,propietario,50",
		"1B,cochera,10,0.5,,,,,,",
		"1C,local,75.25,2,Ana Gomez,,,,,",
	)
	filas, errores, err := parsePlantillaUFSCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parsePlantillaUFSCSV: unexpected error %v", err)
	}
	if len(errores) != 0 {
		t.Fatalf("se esperaban 0 errores, hay %d: %+v", len(errores), errores)
	}
	if len(filas) != 3 {
		t.Fatalf("se esperaban 3 filas, hay %d", len(filas))
	}
	f := filas[0]
	if f.Fila != 2 || f.Codigo != "1A" || f.Tipo != "departamento" || *f.Superficie != 45.5 || f.Coeficiente != "1.25" {
		t.Fatalf("fila 0 mal parseada: %+v", f)
	}
	if !f.Valida || !f.TienePersona() {
		t.Fatalf("fila 0 debería validar con persona: %+v", f)
	}
	if f.Porcentaje == nil || *f.Porcentaje != "50" {
		t.Fatalf("fila 0 porcentaje no parseado: %+v", f)
	}
	if filas[1].TienePersona() {
		t.Fatalf("fila 1 (sin persona) no debería tener vínculo: %+v", filas[1])
	}
	if filas[1].Superficie == nil || *filas[1].Superficie != 10 {
		t.Fatalf("fila 1 superficie mal parseada: %+v", filas[1])
	}
	if filas[2].Documento != nil {
		t.Fatalf("documento nulo no respetado: %+v", filas[2])
	}
}

func TestParsePlantillaUFSCSVHeader(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"sin encabezado", "1A,tipo_dpto,45.5,1.25,,\n"},
		{"columna faltante", "codigo,tipo,superficie,coeficiente\n"},
		{"orden distinto", "codigo,tipo,superficie,coeficiente,nombre,vinculo,porcentaje,email,telefono,documento\n"},
		{"encabezado erroneo", "codigo,tipo,superficie,coef,nombre,documento,email,telefono,vinculo,porcentaje\n"},
	}
	for _, tc := range tests {
		_, _, err := parsePlantillaUFSCSV(strings.NewReader(tc.in))
		if err == nil || !errors.Is(err, ErrImportInvalid) {
			t.Errorf("%s: se esperaba ErrImportInvalid, got %v", tc.name, err)
		}
	}
}

func TestParsePlantillaUFSCSVBOM(t *testing.T) {
	_, _, err := parsePlantillaUFSCSV(strings.NewReader("\ufeff" + csvHeader + "1A,departamento,45.5,1.25,,,,,,\n"))
	if err != nil {
		t.Fatalf("BOM en encabezado debería tolerarse: %v", err)
	}
}

func TestParsePlantillaUFSCSVErroresPorFila(t *testing.T) {
	body := csvPlants(
		"1A,departamento,45.5,1.25,Maria,,,,,",
		"1A,parcela,-5,-1.123456789,x,,,,,",
		"1B,cochera,10,2,Zoe,,zoe@correo.com,,apoderado,101",
	)
	filas, errores, err := parsePlantillaUFSCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if len(errores) != 2 {
		t.Fatalf("se esperaban 2 filas con error, hay %d: %+v", len(errores), errores)
	}
	// Fila 3: codigo duplicado + tipo inválido + superficie negativa + coeficiente inválido.
	dup := errores[0]
	if dup.Fila != 3 || len(dup.Campos) != 4 {
		t.Fatalf("errores fila duplicada: %+v", dup)
	}
	// Fila 4: porcentaje > 100.
	badPct := errores[1]
	if badPct.Fila != 4 || len(badPct.Campos) != 1 || badPct.Campos[0].Campo != "porcentaje" {
		t.Fatalf("errores fila porcentaje: %+v", badPct)
	}
	if !filas[0].Valida || filas[1].Valida || filas[2].Valida {
		t.Fatalf("flags Valida incorrectos: %+v", filas)
	}
}

func TestParsePlantillaUFSCSVPersonaSinNombre(t *testing.T) {
	body := csvPlants("1A,departamento,45.5,1.25,,DNI 30111222,,,inquilino,30")
	_, errores, err := parsePlantillaUFSCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if len(errores) != 1 {
		t.Fatalf("se esperaba 1 error (nombre obligatorio), hay %d: %+v", len(errores), errores)
	}
	foundNombre := false
	for _, c := range errores[0].Campos {
		if c.Campo == "nombre" {
			foundNombre = true
			break
		}
	}
	if !foundNombre {
		t.Fatalf("error esperado en campo nombre, got %+v", errores[0].Campos)
	}
}

func TestParsePlantillaUFSCSVArchivoVacio(t *testing.T) {
	_, _, err := parsePlantillaUFSCSV(strings.NewReader(csvHeader))
	if err == nil || !errors.Is(err, ErrImportInvalid) {
		t.Fatalf("solo encabezado debería fallar, got %v", err)
	}
	_, _, err = parsePlantillaUFSCSV(strings.NewReader(""))
	if err == nil || !errors.Is(err, ErrImportInvalid) {
		t.Fatalf("archivo vacío debería fallar, got %v", err)
	}
}

func TestPlantillaUFSCSVEncabezado(t *testing.T) {
	if len(encabezadoUFSCSV) != 10 {
		t.Fatalf("el encabezado de plantilla debe tener 10 columnas")
	}
}