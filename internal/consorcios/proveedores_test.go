package consorcios

import (
	"errors"
	"testing"
)

func TestValidateProveedor(t *testing.T) {
	t.Run("valido", func(t *testing.T) {
		v, err := validateProveedor(ProveedorInput{
			Cuit:             "30500000011",
			RazonSocial:      "Electricidad SRL",
			ContactoNombre:   ptr("Jorge"),
			ContactoEmail:    ptr(" jorge@mail.com "),
			ContactoTelefono: ptr("1155552222"),
		}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.cuit != "30500000011" || v.razonSocial != "Electricidad SRL" {
			t.Fatalf("cuit/razon_social mal normalizados: %+v", v)
		}
		if !v.contactoEmail.Valid || v.contactoEmail.String != "jorge@mail.com" {
			t.Fatalf("contacto_email debería normalizarse (trim): %+v", v.contactoEmail)
		}
		if !v.contactoNombre.Valid || v.contactoNombre.String != "Jorge" {
			t.Fatalf("contacto_nombre debería normalizarse: %+v", v.contactoNombre)
		}
	})
	t.Run("cuit vacio/toggle con trim", func(t *testing.T) {
		if _, err := validateProveedor(ProveedorInput{Cuit: "  30500000011  ", RazonSocial: "x"}, true); err != nil {
			t.Fatalf("cuit con espacios debería validar tras trim: %v", err)
		}
		if _, err := validateProveedor(ProveedorInput{Cuit: "", RazonSocial: "x"}, true); !errors.Is(err, ErrProveedorInvalid) {
			t.Fatalf("cuit vacío debería fallar, got %v", err)
		}
	})
	t.Run("cuit invalido", func(t *testing.T) {
		for _, c := range []string{"123", "abc12345678", "305000000111", "3050000001a"} {
			if _, err := validateProveedor(ProveedorInput{Cuit: c, RazonSocial: "x"}, true); !errors.Is(err, ErrProveedorInvalid) {
				t.Errorf("cuit %q: want ErrProveedorInvalid, got %v", c, err)
			}
		}
	})
	t.Run("razon_social requerida", func(t *testing.T) {
		if _, err := validateProveedor(ProveedorInput{Cuit: "30500000011", RazonSocial: ""}, true); !errors.Is(err, ErrProveedorInvalid) {
			t.Fatalf("razon_social vacía debería fallar, got %v", err)
		}
	})
	t.Run("contactos opcionales null", func(t *testing.T) {
		v, err := validateProveedor(ProveedorInput{Cuit: "30500000011", RazonSocial: "x", ContactoEmail: ptr("")}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.contactoEmail.Valid {
			t.Fatalf("email vacío debería ser null: %+v", v.contactoEmail)
		}
	})
}

func TestValidProveedorEstados(t *testing.T) {
	if !validProveedorEstados["activo"] || !validProveedorEstados["inactivo"] {
		t.Fatalf("estados de proveedor esperados: activo/inactivo")
	}
	if len(validProveedorEstados) != 2 {
		t.Fatalf("no deberían existir otros estados")
	}
}
