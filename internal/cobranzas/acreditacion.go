package cobranzas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrCobranzaEstadoInvalido = errors.New("estado de cobranza inválido")
	ErrAsignacionesInvalidas  = errors.New("asignaciones inválidas")
	ErrCobranzaNoRevertible   = errors.New("cobranza no reversible")
)

const (
	idempotencyScopeAcreditar    = "cobranzas.acreditar"
	idempotencyScopeAsignaciones = "cobranzas.asignaciones"
	idempotencyScopeRevertir     = "cobranzas.revertir"
)

type AllocationInput struct {
	ChargeID    string `json:"charge_id"`
	AmountCents int64  `json:"amount_cents"`
}

type AllocationDTO struct {
	ChargeID    string `json:"charge_id"`
	AmountCents int64  `json:"amount_cents"`
}

type AcreditacionDTO struct {
	Cobranza         CobranzaDTO     `json:"cobranza"`
	Asignaciones     []AllocationDTO `json:"asignaciones"`
	SaldoAFavorCents int64           `json:"saldo_a_favor_cents"`
}

type chargeCandidate struct {
	ID         string
	DueDate    time.Time
	SaldoCents int64
	CreatedAt  time.Time
	UnidadID   string
}

type allocationPlan struct {
	ChargeID    string
	AmountCents int64
}

func AcreditarCobranza(ctx context.Context, q *db.Queries, paymentID, actorID string, requested []AllocationInput, idemKey string) (AcreditacionDTO, error) {
	mode := "fifo"
	if requested != nil {
		mode = "manual"
	}
	if idemKey != "" {
		cached, hit, err := stampOperationIdempotency(ctx, q, idempotencyScopeAcreditar, idemKey, paymentID, mode, requested)
		if err != nil {
			return AcreditacionDTO{}, err
		}
		if hit {
			return cached, nil
		}
	}

	res, err := applyPaymentAllocations(ctx, q, paymentID, actorID, requested, true)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	if idemKey != "" {
		if err := saveOperationIdempotency(ctx, q, idempotencyScopeAcreditar, idemKey, res); err != nil {
			return AcreditacionDTO{}, err
		}
	}
	return res, nil
}

func AjustarAsignaciones(ctx context.Context, q *db.Queries, paymentID, actorID string, requested []AllocationInput, idemKey string) (AcreditacionDTO, error) {
	if idemKey != "" {
		cached, hit, err := stampOperationIdempotency(ctx, q, idempotencyScopeAsignaciones, idemKey, paymentID, "manual", requested)
		if err != nil {
			return AcreditacionDTO{}, err
		}
		if hit {
			return cached, nil
		}
	}

	res, err := applyPaymentAllocations(ctx, q, paymentID, actorID, requested, false)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	if idemKey != "" {
		if err := saveOperationIdempotency(ctx, q, idempotencyScopeAsignaciones, idemKey, res); err != nil {
			return AcreditacionDTO{}, err
		}
	}
	return res, nil
}

func RevertirCobranza(ctx context.Context, q *db.Queries, paymentID, actorID string, idemKey string) (AcreditacionDTO, error) {
	if idemKey != "" {
		cached, hit, err := stampOperationIdempotency(ctx, q, idempotencyScopeRevertir, idemKey, paymentID, "revertir", nil)
		if err != nil {
			return AcreditacionDTO{}, err
		}
		if hit {
			return cached, nil
		}
	}

	res, err := revertPayment(ctx, q, paymentID, actorID)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	if idemKey != "" {
		if err := saveOperationIdempotency(ctx, q, idempotencyScopeRevertir, idemKey, res); err != nil {
			return AcreditacionDTO{}, err
		}
	}
	return res, nil
}

func applyPaymentAllocations(ctx context.Context, q *db.Queries, paymentID, actorID string, requested []AllocationInput, markAcreditado bool) (AcreditacionDTO, error) {
	var pid, aid pgtype.UUID
	if err := pid.Scan(strings.TrimSpace(paymentID)); err != nil {
		return AcreditacionDTO{}, ErrCobranzaNotFound
	}
	if err := aid.Scan(strings.TrimSpace(actorID)); err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}

	payment, err := q.GetCobranza(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return AcreditacionDTO{}, ErrCobranzaNotFound
	}
	if err != nil {
		return AcreditacionDTO{}, err
	}
	if markAcreditado && payment.Estado != "pendiente_revision" {
		return AcreditacionDTO{}, ErrCobranzaEstadoInvalido
	}
	if !markAcreditado && payment.Estado != "acreditado" {
		return AcreditacionDTO{}, ErrCobranzaEstadoInvalido
	}

	// Carga de cargos elegibles: antes de aplicar una acreditación usamos FIFO;
	// en ajustes, restauramos primero el estado previo y volvemos a leer.
	if !markAcreditado {
		if err := restoreExistingAllocations(ctx, q, pid); err != nil {
			return AcreditacionDTO{}, err
		}
	}

	charges, err := q.ListOpenChargesByUnidad(ctx, db.ListOpenChargesByUnidadParams{
		UnidadID:   payment.UnidadID,
		FechaCorte: payment.Fecha,
	})
	if err != nil {
		return AcreditacionDTO{}, err
	}
	candidates := make([]chargeCandidate, 0, len(charges))
	for _, ch := range charges {
		candidates = append(candidates, chargeCandidate{
			ID:         ch.ID.String(),
			DueDate:    ch.DueDate.Time,
			SaldoCents: ch.SaldoCents,
			CreatedAt:  ch.CreatedAt.Time,
			UnidadID:   ch.UnidadID.String(),
		})
	}

	var plans []allocationPlan
	var saldoAFavor int64
	if requested == nil {
		plans, saldoAFavor = proposeFIFOAllocations(payment.Fecha.Time, payment.ImporteCents, candidates)
	} else {
		plans, saldoAFavor, err = validateManualAllocations(payment.ImporteCents, candidates, requested)
		if err != nil {
			return AcreditacionDTO{}, err
		}
	}

	if markAcreditado {
		if err := insertCreditEntry(ctx, q, payment, aid); err != nil {
			return AcreditacionDTO{}, err
		}
	}

	if err := persistAllocations(ctx, q, payment, aid, plans); err != nil {
		return AcreditacionDTO{}, err
	}
	if err := decrementChargeBalances(ctx, q, plans); err != nil {
		return AcreditacionDTO{}, err
	}
	if markAcreditado {
		if err := q.UpdateCobranzaEstado(ctx, db.UpdateCobranzaEstadoParams{ID: pid, Estado: "acreditado"}); err != nil {
			return AcreditacionDTO{}, err
		}
	}

	updated, err := q.GetCobranza(ctx, pid)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	allocRows, err := q.ListPaymentAllocationsByPayment(ctx, pid)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	return AcreditacionDTO{
		Cobranza:         cobranzaDTO(updated, allocRows),
		Asignaciones:     allocationRowsToDTO(allocRows),
		SaldoAFavorCents: saldoAFavor,
	}, nil
}

func revertPayment(ctx context.Context, q *db.Queries, paymentID, actorID string) (AcreditacionDTO, error) {
	var pid pgtype.UUID
	if err := pid.Scan(strings.TrimSpace(paymentID)); err != nil {
		return AcreditacionDTO{}, ErrCobranzaNotFound
	}

	payment, err := q.GetCobranza(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return AcreditacionDTO{}, ErrCobranzaNotFound
	}
	if err != nil {
		return AcreditacionDTO{}, err
	}
	if payment.Estado != "acreditado" {
		return AcreditacionDTO{}, ErrCobranzaNoRevertible
	}

	allocs, err := q.ListPaymentAllocationsByPayment(ctx, pid)
	if err != nil {
		return AcreditacionDTO{}, err
	}

	if err := incrementChargeBalances(ctx, q, allocs); err != nil {
		return AcreditacionDTO{}, err
	}
	if err := insertReversalEntry(ctx, q, payment); err != nil {
		return AcreditacionDTO{}, err
	}
	if err := q.UpdateCobranzaEstado(ctx, db.UpdateCobranzaEstadoParams{ID: pid, Estado: "revertido"}); err != nil {
		return AcreditacionDTO{}, err
	}

	updated, err := q.GetCobranza(ctx, pid)
	if err != nil {
		return AcreditacionDTO{}, err
	}
	return AcreditacionDTO{
		Cobranza:         cobranzaDTO(updated, nil),
		Asignaciones:     allocationRowsToDTO(allocs),
		SaldoAFavorCents: 0,
	}, nil
}

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}

func incrementChargeBalances(ctx context.Context, q *db.Queries, allocs []db.PaymentAllocation) error {
	for _, a := range allocs {
		ch, err := q.GetCharge(ctx, a.ChargeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAsignacionesInvalidas
		}
		if err != nil {
			return err
		}
		newSaldo := ch.SaldoCents + a.AmountCents
		if newSaldo > ch.TotalCents {
			return ErrAsignacionesInvalidas
		}
		if err := q.UpdateChargeSaldo(ctx, db.UpdateChargeSaldoParams{ID: ch.ID, SaldoCents: newSaldo}); err != nil {
			return err
		}
	}
	return nil
}

func insertReversalEntry(ctx context.Context, q *db.Queries, payment db.Payment) error {
	referencia := "reversa de cobranza " + payment.ID.String()
	if payment.Referencia.Valid && payment.Referencia.String != "" {
		referencia = referencia + " (" + payment.Referencia.String + ")"
	}
	_, err := q.InsertAccountEntry(ctx, db.InsertAccountEntryParams{
		UnidadID:      payment.UnidadID,
		Tipo:          "reversa",
		FechaEfectiva: payment.Fecha,
		DebitCents:    payment.ImporteCents,
		CreditCents:   0,
		Currency:      "ARS",
		Referencia:    pgtype.Text{String: referencia, Valid: true},
	})
	return err
}

func restoreExistingAllocations(ctx context.Context, q *db.Queries, paymentID pgtype.UUID) error {
	allocs, err := q.ListPaymentAllocationsByPayment(ctx, paymentID)
	if err != nil {
		return err
	}
	for _, a := range allocs {
		ch, err := q.GetCharge(ctx, a.ChargeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAsignacionesInvalidas
		}
		if err != nil {
			return err
		}
		if err := q.UpdateChargeSaldo(ctx, db.UpdateChargeSaldoParams{ID: ch.ID, SaldoCents: ch.SaldoCents + a.AmountCents}); err != nil {
			return err
		}
	}
	if _, err := q.DeletePaymentAllocationsByPayment(ctx, paymentID); err != nil {
		return err
	}
	return nil
}

func insertCreditEntry(ctx context.Context, q *db.Queries, payment db.Payment, actor pgtype.UUID) error {
	referencia := "cobranza " + payment.ID.String()
	if payment.Referencia.Valid && payment.Referencia.String != "" {
		referencia = payment.Referencia.String
	}
	_, err := q.InsertAccountEntry(ctx, db.InsertAccountEntryParams{
		UnidadID:      payment.UnidadID,
		Tipo:          "credito",
		FechaEfectiva: payment.Fecha,
		DebitCents:    0,
		CreditCents:   payment.ImporteCents,
		Currency:      "ARS",
		Referencia:    pgtype.Text{String: referencia, Valid: true},
	})
	return err
}

func persistAllocations(ctx context.Context, q *db.Queries, payment db.Payment, actor pgtype.UUID, plans []allocationPlan) error {
	for _, plan := range plans {
		cid, err := parseUUID(plan.ChargeID)
		if err != nil {
			return ErrAsignacionesInvalidas
		}
		if _, err := q.InsertPaymentAllocation(ctx, db.InsertPaymentAllocationParams{
			PaymentID:   payment.ID,
			ChargeID:    cid,
			AmountCents: plan.AmountCents,
			CreatedBy:   actor,
		}); err != nil {
			return err
		}
	}
	return nil
}

func decrementChargeBalances(ctx context.Context, q *db.Queries, plans []allocationPlan) error {
	for _, plan := range plans {
		cid, err := parseUUID(plan.ChargeID)
		if err != nil {
			return ErrAsignacionesInvalidas
		}
		ch, err := q.GetCharge(ctx, cid)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAsignacionesInvalidas
		}
		if err != nil {
			return err
		}
		newSaldo := ch.SaldoCents - plan.AmountCents
		if newSaldo < 0 || newSaldo > ch.TotalCents {
			return ErrAsignacionesInvalidas
		}
		if err := q.UpdateChargeSaldo(ctx, db.UpdateChargeSaldoParams{ID: ch.ID, SaldoCents: newSaldo}); err != nil {
			return err
		}
	}
	return nil
}

func proposeFIFOAllocations(paymentDate time.Time, importe int64, charges []chargeCandidate) ([]allocationPlan, int64) {
	eligible := make([]chargeCandidate, 0, len(charges))
	for _, ch := range charges {
		if !ch.DueDate.After(paymentDate) && ch.SaldoCents > 0 {
			eligible = append(eligible, ch)
		}
	}
	sort.Slice(eligible, func(i, j int) bool {
		if !eligible[i].DueDate.Equal(eligible[j].DueDate) {
			return eligible[i].DueDate.Before(eligible[j].DueDate)
		}
		if !eligible[i].CreatedAt.Equal(eligible[j].CreatedAt) {
			return eligible[i].CreatedAt.Before(eligible[j].CreatedAt)
		}
		return eligible[i].ID < eligible[j].ID
	})
	remaining := importe
	plans := make([]allocationPlan, 0, len(eligible))
	for _, ch := range eligible {
		if remaining <= 0 {
			break
		}
		applied := ch.SaldoCents
		if applied > remaining {
			applied = remaining
		}
		plans = append(plans, allocationPlan{ChargeID: ch.ID, AmountCents: applied})
		remaining -= applied
	}
	return plans, remaining
}

func validateManualAllocations(importe int64, charges []chargeCandidate, requested []AllocationInput) ([]allocationPlan, int64, error) {
	byID := make(map[string]chargeCandidate, len(charges))
	for _, ch := range charges {
		byID[ch.ID] = ch
	}
	seen := map[string]bool{}
	plans := make([]allocationPlan, 0, len(requested))
	var total int64
	for _, req := range requested {
		if req.AmountCents <= 0 {
			return nil, 0, ErrAsignacionesInvalidas
		}
		if seen[req.ChargeID] {
			return nil, 0, ErrAsignacionesInvalidas
		}
		ch, ok := byID[req.ChargeID]
		if !ok {
			return nil, 0, ErrAsignacionesInvalidas
		}
		if req.AmountCents > ch.SaldoCents {
			return nil, 0, ErrAsignacionesInvalidas
		}
		seen[req.ChargeID] = true
		plans = append(plans, allocationPlan(req))
		total += req.AmountCents
	}
	if total > importe {
		return nil, 0, ErrAsignacionesInvalidas
	}
	return plans, importe - total, nil
}

func allocationRowsToDTO(rows []db.PaymentAllocation) []AllocationDTO {
	out := make([]AllocationDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, AllocationDTO{ChargeID: row.ChargeID.String(), AmountCents: row.AmountCents})
	}
	return out
}

func stampOperationIdempotency(ctx context.Context, q *db.Queries, scope, idemKey, paymentID, mode string, requested []AllocationInput) (AcreditacionDTO, bool, error) {
	hash := requestHashOperation(paymentID, mode, requested)
	inserted, err := q.InsertIdempotencyKey(ctx, db.InsertIdempotencyKeyParams{
		IdempotencyKey: idemKey,
		Scope:          scope,
		RequestHash:    hash,
		ResponseJson:   []byte("{}"),
	})
	if err != nil {
		return AcreditacionDTO{}, false, err
	}
	if inserted == 0 {
		existing, err := q.GetIdempotencyKey(ctx, db.GetIdempotencyKeyParams{Scope: scope, IdempotencyKey: idemKey})
		if err != nil {
			return AcreditacionDTO{}, false, err
		}
		if existing.RequestHash != hash {
			return AcreditacionDTO{}, false, ErrIdempotencyConflict
		}
		var cached AcreditacionDTO
		if err := json.Unmarshal(existing.ResponseJson, &cached); err != nil {
			return AcreditacionDTO{}, false, err
		}
		return cached, true, nil
	}
	return AcreditacionDTO{}, false, nil
}

func saveOperationIdempotency(ctx context.Context, q *db.Queries, scope, idemKey string, dto AcreditacionDTO) error {
	resp, err := json.Marshal(dto)
	if err != nil {
		return err
	}
	return q.UpdateIdempotencyKey(ctx, db.UpdateIdempotencyKeyParams{
		Scope:          scope,
		IdempotencyKey: idemKey,
		ResponseJson:   resp,
	})
}

func requestHashOperation(paymentID, mode string, requested []AllocationInput) string {
	canonical := append([]AllocationInput(nil), requested...)
	sort.Slice(canonical, func(i, j int) bool {
		if canonical[i].ChargeID == canonical[j].ChargeID {
			return canonical[i].AmountCents < canonical[j].AmountCents
		}
		return canonical[i].ChargeID < canonical[j].ChargeID
	})
	buf, _ := json.Marshal(canonical)
	h := sha256.Sum256([]byte(strings.Join([]string{paymentID, mode, string(buf)}, "|")))
	return hex.EncodeToString(h[:])
}
