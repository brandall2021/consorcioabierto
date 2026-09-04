package consorcios

import (
	"errors"
	"strings"
	"testing"
)

func TestParseNumeric(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0", "0"},
		{"1", "1"},
		{"0.5", "0.5"},
		{"1.25", "1.25"},
		{"0.25000000", "0.25"},
		{"12345678", "12345678"},
	}
	for _, c := range cases {
		n, err := parseNumeric(c.in)
		if err != nil {
			t.Errorf("parseNumeric(%q): unexpected error %v", c.in, err)
			continue
		}
		if got := numericToString(n); got != c.want {
			t.Errorf("parseNumeric(%q) -> %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseNumericInvalid(t *testing.T) {
	for _, in := range []string{"", "-1", "1.123456789", "abc", "1..2", "0."} {
		if _, err := parseNumeric(in); err == nil {
			t.Errorf("parseNumeric(%q): want error, got nil", in)
		}
	}
}

func TestValidateUnidad(t *testing.T) {
	t.Run("create valido con coeficiente default", func(t *testing.T) {
		v, err := validateUnidad(UnidadInput{Codigo: "  101  ", Tipo: "departamento"}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.codigo != "101" {
			t.Errorf("codigo = %q, want trimmed 101", v.codigo)
		}
		if got := numericToString(v.coeficiente); got != "0" {
			t.Errorf("coeficiente = %q, want 0", got)
		}
	})

	t.Run("create requiere codigo y tipo valido", func(t *testing.T) {
		if _, err := validateUnidad(UnidadInput{}, true); !errors.Is(err, ErrUnidadInvalid) {
			t.Errorf("vacío: want ErrUnidadInvalid, got %v", err)
		}
		if _, err := validateUnidad(UnidadInput{Codigo: "   ", Tipo: "departamento"}, true); !errors.Is(err, ErrUnidadInvalid) {
			t.Errorf("codigo blank: want ErrUnidadInvalid, got %v", err)
		}
		if _, err := validateUnidad(UnidadInput{Codigo: "101", Tipo: "estadio"}, true); !errors.Is(err, ErrUnidadInvalid) {
			t.Errorf("tipo invalido: want ErrUnidadInvalid, got %v", err)
		}
	})

	t.Run("codigo demasiado largo", func(t *testing.T) {
		bad := strings.Repeat("a", 51)
		if _, err := validateUnidad(UnidadInput{Codigo: bad, Tipo: "cochera"}, true); !errors.Is(err, ErrUnidadInvalid) {
			t.Errorf("want ErrUnidadInvalid, got %v", err)
		}
	})

	t.Run("superficie negativa rechazada", func(t *testing.T) {
		neg := -1.0
		if _, err := validateUnidad(UnidadInput{Codigo: "101", Tipo: "local", Superficie: &neg}, true); !errors.Is(err, ErrUnidadInvalid) {
			t.Errorf("want ErrUnidadInvalid, got %v", err)
		}
	})

	t.Run("coeficiente invalido", func(t *testing.T) {
		if _, err := validateUnidad(UnidadInput{Codigo: "101", Tipo: "otros", Coeficiente: "1.5.5"}, true); !errors.Is(err, ErrUnidadInvalid) {
			t.Errorf("want ErrUnidadInvalid, got %v", err)
		}
	})
}

func TestValidateVinculo(t *testing.T) {
	base := PersonaVinculoInput{
		Persona: PersonaInput{Nombre: "  Ana Pérez  "},
		Vinculo: "propietario",
	}

	t.Run("valido con defaults", func(t *testing.T) {
		v, err := validateVinculo(base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.nombre != "Ana Pérez" {
			t.Errorf("nombre = %q, want trimmed", v.nombre)
		}
		if !v.validFrom.Valid {
			t.Error("validFrom debe estar seteado por defecto")
		}
	})

	t.Run("requiere nombre y vinculo valido", func(t *testing.T) {
		if _, err := validateVinculo(PersonaVinculoInput{Vinculo: "propietario"}); !errors.Is(err, ErrVinculoInvalid) {
			t.Errorf("sin nombre: want ErrVinculoInvalid, got %v", err)
		}
		if _, err := validateVinculo(PersonaVinculoInput{Persona: PersonaInput{Nombre: "x"}, Vinculo: "dueño"}); !errors.Is(err, ErrVinculoInvalid) {
			t.Errorf("vinculo invalido: want ErrVinculoInvalid, got %v", err)
		}
	})

	t.Run("porcentaje y valid_from validados", func(t *testing.T) {
		ok := base
		ok.Porcentaje = ptr("33.33")
		if _, err := validateVinculo(ok); err != nil {
			t.Errorf("porcentaje valido: unexpected error %v", err)
		}

		bad := base
		bad.Porcentaje = ptr("101")
		if _, err := validateVinculo(bad); !errors.Is(err, ErrVinculoInvalid) {
			t.Errorf("porcentaje 101: want ErrVinculoInvalid, got %v", err)
		}

		badDate := base
		badDate.ValidFrom = ptr("2024-13-40")
		if _, err := validateVinculo(badDate); !errors.Is(err, ErrVinculoInvalid) {
			t.Errorf("valid_from invalido: want ErrVinculoInvalid, got %v", err)
		}
	})
}

func TestFloatToNumeric(t *testing.T) {
	if got := numericToString(floatToNumeric(72.5)); got != "72.5" {
		t.Errorf("floatToNumeric(72.5) -> %q, want 72.5", got)
	}
}