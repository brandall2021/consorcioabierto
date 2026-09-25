package cobranzas

import (
	"strings"
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestGenerateReciboPDFIncludesCobranzaData(t *testing.T) {
	payment := db.Payment{
		ID:           uuidFromString(t, "00000000-0000-0000-0000-000000000101"),
		UnidadID:     uuidFromString(t, "00000000-0000-0000-0000-000000000102"),
		Fecha:        dateFromString(t, "2026-09-10"),
		Canal:        "transferencia",
		ImporteCents: 5000,
		Referencia:   textFromString(t, "TRX-123"),
		Estado:       "acreditado",
		CreatedAt:    tsFromTime(time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)),
	}
	recibo, err := GenerateReciboPDF(CobranzaDTO{ID: payment.ID.String(), UnidadID: payment.UnidadID.String(), Fecha: "2026-09-10", Canal: "transferencia", Importe: MoneyDTO{AmountCents: 5000, Currency: "ARS"}, Referencia: ptr("TRX-123"), Estado: "acreditado", SaldoAFavorCents: 2000}, []AllocationDTO{{ChargeID: "charge-1", AmountCents: 3000}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(string(recibo), "%PDF-1.4") {
		t.Fatal("missing pdf header")
	}
	for _, want := range []string{"Recibo de Cobranza", "5000", "Saldo a favor", "TRX-123"} {
		if !strings.Contains(string(recibo), want) {
			t.Fatalf("pdf missing %q", want)
		}
	}
}

func ptr(s string) *string { return &s }

func uuidFromString(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		t.Fatalf("uuid scan: %v", err)
	}
	return u
}

func dateFromString(t *testing.T, s string) pgtype.Date {
	t.Helper()
	var d pgtype.Date
	if err := d.Scan(s); err != nil {
		t.Fatalf("date scan: %v", err)
	}
	return d
}

func textFromString(t *testing.T, s string) pgtype.Text {
	t.Helper()
	var txt pgtype.Text
	if err := txt.Scan(s); err != nil {
		t.Fatalf("text scan: %v", err)
	}
	return txt
}

func tsFromTime(ti time.Time) pgtype.Timestamptz {
	var ts pgtype.Timestamptz
	_ = ts.Scan(ti)
	return ts
}
