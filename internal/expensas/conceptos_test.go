package expensas

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateConcepto(t *testing.T) {
	t.Run("valido", func(t *testing.T) {
		v, err := validateConcepto(ConceptoInput{Nombre: "Expensas de servicios", Categoria: "servicios"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.nombre != "Expensas de servicios" || v.categoria != "servicios" {
			t.Fatalf("valores mal normalizados: %+v", v)
		}
	})
	t.Run("nombre vacio o largo", func(t *testing.T) {
		for _, n := range []string{"", "   ", strings.Repeat("x", 101)} {
			if _, err := validateConcepto(ConceptoInput{Nombre: n, Categoria: "servicios"}); !errors.Is(err, ErrConceptoInvalid) {
				t.Errorf("nombre %q: want ErrConceptoInvalid, got %v", n, err)
			}
		}
	})
	t.Run("categoria invalida", func(t *testing.T) {
		if _, err := validateConcepto(ConceptoInput{Nombre: "x", Categoria: "inventada"}); !errors.Is(err, ErrConceptoInvalid) {
			t.Fatalf("categoria inventada debería fallar, got %v", err)
		}
	})
	t.Run("trim de espacios", func(t *testing.T) {
		v, err := validateConcepto(ConceptoInput{Nombre: "  Agua  ", Categoria: "servicios"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.nombre != "Agua" {
			t.Fatalf("nombre con espacios debería normalizarse, got %q", v.nombre)
		}
	})
}
