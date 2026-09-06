package expensas

import (
	"errors"
	"testing"
	"time"
)

func TestValidateGasto(t *testing.T) {
	t.Run("valido", func(t *testing.T) {
		v, err := validateGasto(GastoInput{ImporteCents: 1000, Fecha: "2026-09-01"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.importeCents != 1000 {
			t.Fatalf("importe mal: %d", v.importeCents)
		}
	})
	t.Run("importe no positivo", func(t *testing.T) {
		for _, c := range []int64{0, -1} {
			if _, err := validateGasto(GastoInput{ImporteCents: c, Fecha: "2026-09-01"}); !errors.Is(err, ErrGastoInvalid) {
				t.Errorf("importe %d: want ErrGastoInvalid, got %v", c, err)
			}
		}
	})
	t.Run("fecha invalida o futura", func(t *testing.T) {
		for _, f := range []string{"2026/09/01", "01-09-2026", "2026-09-01T00:00:00Z"} {
			if _, err := validateGasto(GastoInput{ImporteCents: 1000, Fecha: f}); !errors.Is(err, ErrGastoInvalid) {
				t.Errorf("fecha %q debería fallar, got %v", f, err)
			}
		}
	})
	t.Run("fecha futura rechazada", func(t *testing.T) {
		f := time.Now().Add(48 * time.Hour).Format("2006-01-02")
		if _, err := validateGasto(GastoInput{ImporteCents: 1, Fecha: f}); !errors.Is(err, ErrGastoInvalid) {
			t.Fatalf("fecha futura debería fallar, got %v", err)
		}
	})
}
