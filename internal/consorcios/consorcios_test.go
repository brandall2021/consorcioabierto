package consorcios

import (
	"errors"
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func TestValidate(t *testing.T) {
	t.Run("create valido con defaults", func(t *testing.T) {
		in := Input{Nombre: ptr("  Torres del Sol  "), Cuit: ptr("30500000011")}
		v, err := validate(in, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.nombre != "Torres del Sol" {
			t.Errorf("nombre = %q, want trimmed", v.nombre)
		}
		if v.tipo != "edificio" {
			t.Errorf("tipo = %q, want default edificio", v.tipo)
		}
		if v.cuit != "30500000011" {
			t.Errorf("cuit = %q", v.cuit)
		}
	})

	t.Run("create requiere nombre", func(t *testing.T) {
		if _, err := validate(Input{}, true); !errors.Is(err, ErrInvalid) {
			t.Errorf("want ErrInvalid, got %v", err)
		}
		if _, err := validate(Input{Nombre: ptr("   ")}, true); !errors.Is(err, ErrInvalid) {
			t.Errorf("want ErrInvalid for blank nombre, got %v", err)
		}
	})

	t.Run("nombre demasiado largo", func(t *testing.T) {
		bad := strings.Repeat("a", 201)
		if _, err := validate(Input{Nombre: ptr(bad)}, true); !errors.Is(err, ErrInvalid) {
			t.Errorf("want ErrInvalid, got %v", err)
		}
	})

	t.Run("cuit invalido", func(t *testing.T) {
		for _, cuit := range []string{"123", "abc12345678", "305000000111", ""} {
			in := Input{Nombre: ptr("x"), Cuit: ptr(cuit)}
			if _, err := validate(in, true); !errors.Is(err, ErrInvalid) {
				t.Errorf("cuit %q: want ErrInvalid, got %v", cuit, err)
			}
		}
	})

	t.Run("tipo y estado invalidos", func(t *testing.T) {
		if _, err := validate(Input{Nombre: ptr("x"), Tipo: ptr("estadio")}, true); !errors.Is(err, ErrInvalid) {
			t.Errorf("tipo invalido: want ErrInvalid, got %v", err)
		}
		if _, err := validate(Input{Nombre: ptr("x"), Tipo: ptr("barrio")}, true); err != nil {
			t.Errorf("tipo barrio valido: unexpected error %v", err)
		}
	})

	t.Run("update parcial sin nombre requerido", func(t *testing.T) {
		v, err := validate(Input{Tipo: ptr("complejo")}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.tipo != "complejo" {
			t.Errorf("tipo = %q", v.tipo)
		}
		if v.nombre != "" {
			t.Errorf("nombre = %q, want empty (no requerido en patch)", v.nombre)
		}
	})
}

func TestValidateUpdate(t *testing.T) {
	// Update usa checkNombre por campo: nombre vacío tras trim es inválido.
	if err := checkNombre("   "); !errors.Is(err, ErrInvalid) {
		t.Errorf("blank nombre: want ErrInvalid, got %v", err)
	}
	if err := checkNombre(" Torres "); err != nil {
		t.Errorf("valid nombre: unexpected error %v", err)
	}
}
