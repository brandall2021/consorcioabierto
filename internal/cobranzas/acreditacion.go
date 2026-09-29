package cobranzas

import (
	"context"
	"errors"
	"fmt"
	"strings"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AsignacionInput struct {
	ChargeID    string `json:"charge_id"`
	AmountCents int64  `json:"amount_cents"`
}

type AsignacionDTO struct {
	ChargeID    string `json:"charge_id"`
	AmountCents int64  `json:"amount_cents"`
}

type AcreditacionDTO struct {
	Cobranza         CobranzaDTO     `json:"cobranza"`
	Asignaciones     []AsignacionDTO `json:"asignaciones"`
	SaldoAFavorCents int64           `json:"saldo_a_favor_cents"`
}

func allocationRowsToDTO(rows []db.PaymentAllocation) []AsignacionDTO {
	out := make([]AsignacionDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, AsignacionDTO{
			ChargeID:    row.ChargeID.String(),
			AmountCents: row.AmountCents,
		})
	}
	return out
}

func parseUUID(value string) (pgtype.UUID, error) {
	var out pgtype.UUID
	if err := out.Scan(strings.TrimSpace(value)); err != nil {
		return pgtype.UUID{}, err
	}
	return out, nil
}

func AcreditarCobranza(ctx context.Context, q *db.Queries, paymentID, createdBy string, asignaciones []AsignacionInput, _ string) (AcreditacionDTO, error) {
	pid, err := parseUUID(paymentID)
	if err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}
	actor, err := parseUUID(createdBy)
	if err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}

	payment, err := q.GetCobranza(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return AcreditacionDTO{}, ErrCobranzaNotFound
	}
	if err != nil {
		return AcreditacionDTO{}, err
	}

	if payment.Estado == "acreditado" {
		allocs, err := q.ListPaymentAllocationsByPayment(ctx, pid)
		if err != nil {
			return AcreditacionDTO{}, err
		}
		return AcreditacionDTO{Cobranza: cobranzaDTO(payment, allocs), Asignaciones: allocationRowsToDTO(allocs), SaldoAFavorCents: saldoAFavorFromPayment(payment, allocs)}, nil
	}

	allocs, saldoAFavor, err := buildAllocations(ctx, q, payment, asignaciones, actor)
	if err != nil {
		return AcreditacionDTO{}, err
	}

	if err := q.UpdateCobranzaEstado(ctx, db.UpdateCobranzaEstadoParams{Estado: "acreditado", MotivoRechazo: pgtype.Text{}, ID: pid}); err != nil {
		return AcreditacionDTO{}, err
	}

	updated, err := q.GetCobranza(ctx, pid)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	return AcreditacionDTO{Cobranza: cobranzaDTO(updated, allocs), Asignaciones: allocationRowsToDTO(allocs), SaldoAFavorCents: saldoAFavor}, nil
}

func RevertirCobranza(ctx context.Context, q *db.Queries, paymentID, createdBy, idemKey string) (AcreditacionDTO, error) {
	_ = idemKey
	pid, err := parseUUID(paymentID)
	if err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}
	if _, err := parseUUID(createdBy); err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}

	_, err = q.GetCobranza(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return AcreditacionDTO{}, ErrCobranzaNotFound
	}
	if err != nil {
		return AcreditacionDTO{}, err
	}

	allocs, err := q.ListPaymentAllocationsByPayment(ctx, pid)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	if _, err := q.DeletePaymentAllocationsByPayment(ctx, pid); err != nil {
		return AcreditacionDTO{}, err
	}
	if err := q.UpdateCobranzaEstado(ctx, db.UpdateCobranzaEstadoParams{Estado: "revertido", MotivoRechazo: pgtype.Text{}, ID: pid}); err != nil {
		return AcreditacionDTO{}, err
	}

	updated, err := q.GetCobranza(ctx, pid)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	return AcreditacionDTO{Cobranza: cobranzaDTO(updated, nil), Asignaciones: allocationRowsToDTO(allocs), SaldoAFavorCents: 0}, nil
}

func AjustarAsignaciones(ctx context.Context, q *db.Queries, paymentID, createdBy string, asignaciones []AsignacionInput, _ string) (AcreditacionDTO, error) {
	pid, err := parseUUID(paymentID)
	if err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}
	actor, err := parseUUID(createdBy)
	if err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}

	payment, err := q.GetCobranza(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return AcreditacionDTO{}, ErrCobranzaNotFound
	}
	if err != nil {
		return AcreditacionDTO{}, err
	}

	if _, err := q.DeletePaymentAllocationsByPayment(ctx, pid); err != nil {
		return AcreditacionDTO{}, err
	}
	allocs, saldoAFavor, err := buildExplicitAllocations(ctx, q, payment, asignaciones, actor)
	if err != nil {
		return AcreditacionDTO{}, err
	}

	if payment.Estado != "acreditado" {
		if err := q.UpdateCobranzaEstado(ctx, db.UpdateCobranzaEstadoParams{Estado: "acreditado", MotivoRechazo: pgtype.Text{}, ID: pid}); err != nil {
			return AcreditacionDTO{}, err
		}
		payment.Estado = "acreditado"
	}

	return AcreditacionDTO{Cobranza: cobranzaDTO(payment, allocs), Asignaciones: allocationRowsToDTO(allocs), SaldoAFavorCents: saldoAFavor}, nil
}

func buildAllocations(ctx context.Context, q *db.Queries, payment db.Payment, explicit []AsignacionInput, actor pgtype.UUID) ([]db.PaymentAllocation, int64, error) {
	if len(explicit) > 0 {
		return buildExplicitAllocations(ctx, q, payment, explicit, actor)
	}
	return buildFIFOAllocations(ctx, q, payment, actor)
}

func buildFIFOAllocations(ctx context.Context, q *db.Queries, payment db.Payment, actor pgtype.UUID) ([]db.PaymentAllocation, int64, error) {
	charges, err := q.ListOpenChargesByUnidad(ctx, db.ListOpenChargesByUnidadParams{UnidadID: payment.UnidadID, FechaCorte: payment.Fecha})
	if err != nil {
		return nil, 0, err
	}
	remaining := payment.ImporteCents
	allocs := make([]db.PaymentAllocation, 0, len(charges))
	for _, charge := range charges {
		if remaining <= 0 {
			break
		}
		amount := charge.SaldoCents
		if amount > remaining {
			amount = remaining
		}
		if amount <= 0 {
			continue
		}
		alloc, err := q.InsertPaymentAllocation(ctx, db.InsertPaymentAllocationParams{PaymentID: payment.ID, ChargeID: charge.ID, AmountCents: amount, CreatedBy: actor})
		if err != nil {
			return nil, 0, err
		}
		allocs = append(allocs, alloc)
		remaining -= amount
	}
	return allocs, remaining, nil
}

func buildExplicitAllocations(ctx context.Context, q *db.Queries, payment db.Payment, explicit []AsignacionInput, actor pgtype.UUID) ([]db.PaymentAllocation, int64, error) {
	if len(explicit) == 0 {
		return nil, payment.ImporteCents, nil
	}
	var total int64
	allocs := make([]db.PaymentAllocation, 0, len(explicit))
	for _, in := range explicit {
		chargeID, err := parseUUID(in.ChargeID)
		if err != nil {
			return nil, 0, ErrCobranzaInvalid
		}
		if in.AmountCents <= 0 {
			return nil, 0, ErrCobranzaInvalid
		}
		alloc, err := q.InsertPaymentAllocation(ctx, db.InsertPaymentAllocationParams{PaymentID: payment.ID, ChargeID: chargeID, AmountCents: in.AmountCents, CreatedBy: actor})
		if err != nil {
			return nil, 0, err
		}
		allocs = append(allocs, alloc)
		total += in.AmountCents
	}
	if total > payment.ImporteCents {
		return nil, 0, fmt.Errorf("%w: la suma asignada supera el importe", ErrCobranzaInvalid)
	}
	return allocs, payment.ImporteCents - total, nil
}
