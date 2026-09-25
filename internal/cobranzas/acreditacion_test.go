package cobranzas

import (
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
)

func TestProposeFIFOAllocations(t *testing.T) {
	paymentDate := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	charges := []chargeCandidate{
		{ID: "c3", DueDate: paymentDate.AddDate(0, 0, -1), SaldoCents: 2000, CreatedAt: paymentDate.Add(-3 * time.Hour)},
		{ID: "c1", DueDate: paymentDate.AddDate(0, 0, -5), SaldoCents: 3000, CreatedAt: paymentDate.Add(-2 * time.Hour)},
		{ID: "c2", DueDate: paymentDate.AddDate(0, 0, -5), SaldoCents: 1000, CreatedAt: paymentDate.Add(-1 * time.Hour)},
		{ID: "future", DueDate: paymentDate.AddDate(0, 0, 1), SaldoCents: 9999, CreatedAt: paymentDate},
	}
	plans, saldo := proposeFIFOAllocations(paymentDate, 4000, charges)
	if saldo != 0 {
		t.Fatalf("saldo inesperado: %d", saldo)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}
	if plans[0].ChargeID != "c1" || plans[0].AmountCents != 3000 {
		t.Fatalf("first plan wrong: %+v", plans[0])
	}
	if plans[1].ChargeID != "c2" || plans[1].AmountCents != 1000 {
		t.Fatalf("second plan wrong: %+v", plans[1])
	}
}

func TestValidateManualAllocations(t *testing.T) {
	charges := []chargeCandidate{
		{ID: "c1", SaldoCents: 3000},
		{ID: "c2", SaldoCents: 2000},
	}
	plans, saldo, err := validateManualAllocations(4000, charges, []AllocationInput{
		{ChargeID: "c2", AmountCents: 1000},
		{ChargeID: "c1", AmountCents: 2000},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saldo != 1000 {
		t.Fatalf("saldo unexpected: %d", saldo)
	}
	if len(plans) != 2 || plans[0].ChargeID != "c2" || plans[1].ChargeID != "c1" {
		t.Fatalf("plans wrong: %+v", plans)
	}
}

func TestValidateManualAllocationsRejectsInvalid(t *testing.T) {
	charges := []chargeCandidate{{ID: "c1", SaldoCents: 3000}}
	bad := [][]AllocationInput{
		{{ChargeID: "c1", AmountCents: 4000}},
		{{ChargeID: "c2", AmountCents: 1000}},
		{{ChargeID: "c1", AmountCents: 0}},
		{{ChargeID: "c1", AmountCents: 1000}, {ChargeID: "c1", AmountCents: 1000}},
	}
	for i, in := range bad {
		if _, _, err := validateManualAllocations(4000, charges, in); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestSaldoAFavorFromPayment(t *testing.T) {
	payment := db.Payment{ImporteCents: 5000, Estado: "acreditado"}
	allocs := []db.PaymentAllocation{{AmountCents: 2000}, {AmountCents: 1000}}
	if got := saldoAFavorFromPayment(payment, allocs); got != 2000 {
		t.Fatalf("saldo_afavor unexpected: %d", got)
	}

	payment.Estado = "revertido"
	if got := saldoAFavorFromPayment(payment, allocs); got != 0 {
		t.Fatalf("saldo_afavor for reverted expected 0, got %d", got)
	}
}

func TestCobranzaDTOIncludesSaldoAFavor(t *testing.T) {
	payment := db.Payment{Estado: "acreditado", ImporteCents: 4000}
	allocs := []db.PaymentAllocation{{AmountCents: 1500}}
	dto := cobranzaDTO(payment, allocs)
	if dto.SaldoAFavorCents != 2500 {
		t.Fatalf("saldo_afavor unexpected: %d", dto.SaldoAFavorCents)
	}
	if dto.Importe.AmountCents != 4000 {
		t.Fatalf("importe unexpected: %d", dto.Importe.AmountCents)
	}
}
