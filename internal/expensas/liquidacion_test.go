package expensas

import (
	"testing"
)

func TestValidTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"borrador", "calculada", true},
		{"borrador", "anulada", true},
		{"borrador", "confirmada", false},
		{"calculada", "borrador", true},
		{"calculada", "confirmada", true},
		{"calculada", "anulada", true},
		{"confirmada", "publicada", true},
		{"confirmada", "anulada", true},
		{"publicada", "cerrada", true},
		{"cerrada", "borrador", false},
		{"anulada", "borrador", false},
	}
	for _, tc := range cases {
		got := canTransition(tc.from, tc.to)
		if got != tc.want {
			t.Errorf("canTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestValidatePeriodo(t *testing.T) {
	if err := validatePeriodo("202609"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := validatePeriodo("20269"); err == nil {
		t.Error("expected error for short periodo")
	}
	if err := validatePeriodo("abc"); err == nil {
		t.Error("expected error for non-numeric periodo")
	}
}

func TestValidateVencimientos(t *testing.T) {
	if err := ValidateVencimientos("2026-10-01", nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := ValidateVencimientos("2026-10-01", strPtr("2026-10-15")); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := ValidateVencimientos("2026-10-15", strPtr("2026-10-01")); err == nil {
		t.Error("expected error when vencimiento_2 < vencimiento_1")
	}
}

func strPtr(s string) *string { return &s }
