package outbox

import "testing"

func TestTituloYCuerpoLiquidacionDerivanDelPeriodo(t *testing.T) {
	if got := tituloLiquidacion("2026-09"); got != "Liquidación 2026-09" {
		t.Fatalf("titulo inesperado: %q", got)
	}
	if got := cuerpoLiquidacion("2026-09"); got != "Período 2026-09. Ya está disponible en el portal." {
		t.Fatalf("cuerpo inesperado: %q", got)
	}
}

func TestTituloYCuerpoLiquidacionSinPeriodo(t *testing.T) {
	// Un payload viejo (previo al aviso) puede traer el periodo vacío: el aviso
	// igual debe salir con texto válido y no romper el worker.
	if got := tituloLiquidacion("   "); got != "Liquidación publicada" {
		t.Fatalf("titulo inesperado: %q", got)
	}
	if got := cuerpoLiquidacion(""); got != "Ya está disponible en el portal." {
		t.Fatalf("cuerpo inesperado: %q", got)
	}
}
