package cobranzas

import (
	"errors"
	"testing"
)

func TestValidateCobranzaInput(t *testing.T) {
	in := CobranzaInput{
		UnidadID: "550e8400-e29b-41d4-a716-446655440000",
		Fecha:    "2026-09-08",
		Importe:  MoneyInput{AmountCents: 12500, Currency: "ars"},
	}
	v, err := validateCobranzaInput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.canal != "otros" {
		t.Fatalf("canal por defecto incorrecto: %q", v.canal)
	}
	if v.currency != "ARS" {
		t.Fatalf("currency incorrecta: %q", v.currency)
	}
	if v.importe != 12500 {
		t.Fatalf("importe incorrecto: %d", v.importe)
	}
}

func TestValidateCobranzaInputInvalid(t *testing.T) {
	for _, tc := range []CobranzaInput{
		{},
		{UnidadID: "x", Fecha: "2026-09-08", Importe: MoneyInput{AmountCents: 0, Currency: "ARS"}},
		{UnidadID: "x", Fecha: "2026-09-08", Importe: MoneyInput{AmountCents: 10, Currency: "USD"}},
		{UnidadID: "x", Fecha: "bad", Importe: MoneyInput{AmountCents: 10, Currency: "ARS"}},
	} {
		if _, err := validateCobranzaInput(tc); !errors.Is(err, ErrCobranzaInvalid) {
			t.Fatalf("expected ErrCobranzaInvalid, got %v", err)
		}
	}
}

func TestNormalizeRef(t *testing.T) {
	if normalizeRef(nil) != nil {
		t.Fatalf("nil reference should stay nil")
	}
	ref := "  ABC-123  "
	out := normalizeRef(&ref)
	if out == nil || *out != "ABC-123" {
		t.Fatalf("unexpected normalized reference: %#v", out)
	}
	blank := "   "
	if normalizeRef(&blank) != nil {
		t.Fatalf("blank reference should become nil")
	}
}
