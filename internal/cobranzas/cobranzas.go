package cobranzas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrCobranzaInvalid             = errors.New("datos de cobranza inválidos")
	ErrCobranzaNotFound            = errors.New("cobranza no encontrada")
	ErrCobranzaDuplicateReferencia = errors.New("referencia de cobranza duplicada")
	ErrIdempotencyConflict         = errors.New("Idempotency-Key ya utilizada con otro request")
)

const idempotencyScope = "cobranzas.create"

type MoneyInput struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

type CobranzaInput struct {
	UnidadID   string     `json:"unidad_id"`
	Fecha      string     `json:"fecha"`
	Canal      *string    `json:"canal"`
	Importe    MoneyInput `json:"importe"`
	Referencia *string    `json:"referencia"`
}

type MoneyDTO struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

type CobranzaDTO struct {
	ID               string    `json:"id"`
	UnidadID         string    `json:"unidad_id"`
	Fecha            string    `json:"fecha"`
	Canal            string    `json:"canal"`
	Importe          MoneyDTO  `json:"importe"`
	Referencia       *string   `json:"referencia"`
	Estado           string    `json:"estado"`
	SaldoAFavorCents int64     `json:"saldo_a_favor_cents"`
	PSPProvider      *string   `json:"psp_provider"`
	PSPPreferenceID  *string   `json:"psp_preference_id"`
	PSPCheckoutURL   *string   `json:"psp_checkout_url"`
	CreatedAt        time.Time `json:"created_at"`
}

type validatedCobranza struct {
	unidadID   pgtype.UUID
	fecha      pgtype.Date
	canal      string
	importe    int64
	referencia *string
	currency   string
}

var validCanales = map[string]bool{
	"efectivo":      true,
	"transferencia": true,
	"deposito":      true,
	"tarjeta":       true,
	"cajero":        true,
	"mercadopago":   true,
	"otros":         true,
}

func validateCobranzaInput(in CobranzaInput) (validatedCobranza, error) {
	if strings.TrimSpace(in.UnidadID) == "" {
		return validatedCobranza{}, ErrCobranzaInvalid
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(in.Fecha))
	if err != nil {
		return validatedCobranza{}, ErrCobranzaInvalid
	}
	if in.Importe.AmountCents <= 0 {
		return validatedCobranza{}, ErrCobranzaInvalid
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Importe.Currency))
	if currency == "" {
		currency = "ARS"
	}
	if currency != "ARS" {
		return validatedCobranza{}, ErrCobranzaInvalid
	}
	canal := "otros"
	if in.Canal != nil {
		canal = strings.ToLower(strings.TrimSpace(*in.Canal))
	}
	if canal == "" {
		canal = "otros"
	}
	if !validCanales[canal] {
		return validatedCobranza{}, ErrCobranzaInvalid
	}
	uid := pgtype.UUID{}
	if err := uid.Scan(strings.TrimSpace(in.UnidadID)); err != nil {
		return validatedCobranza{}, ErrCobranzaInvalid
	}
	var fecha pgtype.Date
	if err := fecha.Scan(t.Format("2006-01-02")); err != nil {
		return validatedCobranza{}, ErrCobranzaInvalid
	}
	return validatedCobranza{
		unidadID:   uid,
		fecha:      fecha,
		canal:      canal,
		importe:    in.Importe.AmountCents,
		referencia: normalizeRef(in.Referencia),
		currency:   currency,
	}, nil
}

func normalizeRef(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func cobranzaDTO(row db.Payment, allocations []db.PaymentAllocation) CobranzaDTO {
	dto := CobranzaDTO{
		ID:               row.ID.String(),
		UnidadID:         row.UnidadID.String(),
		Fecha:            row.Fecha.Time.Format("2006-01-02"),
		Canal:            row.Canal,
		Importe:          MoneyDTO{AmountCents: row.ImporteCents, Currency: "ARS"},
		Referencia:       textPtrOrNil(row.Referencia),
		Estado:           row.Estado,
		SaldoAFavorCents: saldoAFavorFromPayment(row, allocations),
		PSPProvider:      textPtrOrNil(row.PspProvider),
		PSPPreferenceID:  textPtrOrNil(row.PspPreferenceID),
		PSPCheckoutURL:   textPtrOrNil(row.PspCheckoutUrl),
		CreatedAt:        row.CreatedAt.Time,
	}
	return dto
}

func saldoAFavorFromPayment(row db.Payment, allocations []db.PaymentAllocation) int64 {
	if row.Estado != "acreditado" {
		return 0
	}
	var allocated int64
	for _, a := range allocations {
		allocated += a.AmountCents
	}
	remaining := row.ImporteCents - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}

func textPtrOrNil(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func ListCobranzas(ctx context.Context, q *db.Queries, consorcioID string) ([]CobranzaDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return nil, ErrCobranzaInvalid
	}
	rows, err := q.ListCobranzas(ctx, cid)
	if err != nil {
		return nil, err
	}
	out := make([]CobranzaDTO, 0, len(rows))
	for _, row := range rows {
		allocs, err := q.ListPaymentAllocationsByPayment(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, cobranzaDTO(row, allocs))
	}
	return out, nil
}

func GetCobranzaDetalle(ctx context.Context, q *db.Queries, paymentID string) (AcreditacionDTO, error) {
	var pid pgtype.UUID
	if err := pid.Scan(strings.TrimSpace(paymentID)); err != nil {
		return AcreditacionDTO{}, ErrCobranzaInvalid
	}

	row, err := q.GetCobranza(ctx, pid)
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
	return AcreditacionDTO{
		Cobranza:         cobranzaDTO(row, allocs),
		Asignaciones:     allocationRowsToDTO(allocs),
		SaldoAFavorCents: saldoAFavorFromPayment(row, allocs),
	}, nil
}

func CreateCobranza(ctx context.Context, q *db.Queries, consorcioID, createdBy string, in CobranzaInput, idemKey string) (CobranzaDTO, error) {
	validated, err := validateCobranzaInput(in)
	if err != nil {
		return CobranzaDTO{}, err
	}

	var cid, actor pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(consorcioID)); err != nil {
		return CobranzaDTO{}, ErrCobranzaInvalid
	}
	if err := actor.Scan(strings.TrimSpace(createdBy)); err != nil {
		return CobranzaDTO{}, ErrCobranzaInvalid
	}

	if _, err := q.GetUnidad(ctx, db.GetUnidadParams{ConsorcioID: cid, ID: validated.unidadID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CobranzaDTO{}, ErrCobranzaNotFound
		}
		return CobranzaDTO{}, err
	}

	if idemKey != "" {
		cached, hit, err := stampIdempotency(ctx, q, consorcioID, idemKey, validated)
		if err != nil {
			return CobranzaDTO{}, err
		}
		if hit {
			return cached, nil
		}
	}

	row, err := q.CreateCobranza(ctx, db.CreateCobranzaParams{
		UnidadID:     validated.unidadID,
		Fecha:        validated.fecha,
		Canal:        validated.canal,
		ImporteCents: validated.importe,
		Referencia:   maybeText(validated.referencia),
		Estado:       "pendiente_revision",
		IdemKey:      idemKey,
		CreatedBy:    actor,
	})
	if err != nil {
		return CobranzaDTO{}, mapCobranzaError(err)
	}
	dto := cobranzaDTO(row, nil)
	if idemKey != "" {
		if err := saveIdempotency(ctx, q, idemKey, dto); err != nil {
			return CobranzaDTO{}, err
		}
	}
	return dto, nil
}

func maybeText(s *string) pgtype.Text {
	var t pgtype.Text
	if s == nil {
		return t
	}
	_ = t.Scan(*s)
	return t
}

func requestHash(consorcioID, idemKey string, v validatedCobranza) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(consorcioID),
		idemKey,
		v.unidadID.String(),
		v.fecha.Time.Format("2006-01-02"),
		v.canal,
		fmt.Sprintf("%d", v.importe),
		v.currency,
		ptrString(v.referencia),
	}, "|")))
	return hex.EncodeToString(h[:])
}

func stampIdempotency(ctx context.Context, q *db.Queries, consorcioID, idemKey string, v validatedCobranza) (CobranzaDTO, bool, error) {
	hash := requestHash(consorcioID, idemKey, v)
	inserted, err := q.InsertIdempotencyKey(ctx, db.InsertIdempotencyKeyParams{
		IdempotencyKey: idemKey,
		Scope:          idempotencyScope,
		RequestHash:    hash,
		ResponseJson:   []byte("{}"),
	})
	if err != nil {
		return CobranzaDTO{}, false, err
	}
	if inserted == 0 {
		existing, err := q.GetIdempotencyKey(ctx, db.GetIdempotencyKeyParams{Scope: idempotencyScope, IdempotencyKey: idemKey})
		if err != nil {
			return CobranzaDTO{}, false, err
		}
		if existing.RequestHash != hash {
			return CobranzaDTO{}, false, ErrIdempotencyConflict
		}
		var cached CobranzaDTO
		if err := json.Unmarshal(existing.ResponseJson, &cached); err != nil {
			return CobranzaDTO{}, false, err
		}
		return cached, true, nil
	}
	return CobranzaDTO{}, false, nil
}

func saveIdempotency(ctx context.Context, q *db.Queries, idemKey string, dto CobranzaDTO) error {
	resp, err := json.Marshal(dto)
	if err != nil {
		return err
	}
	return q.UpdateIdempotencyKey(ctx, db.UpdateIdempotencyKeyParams{
		Scope:          idempotencyScope,
		IdempotencyKey: idemKey,
		ResponseJson:   resp,
	})
}

func mapCobranzaError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "payments_referencia_unique_idx":
			return ErrCobranzaDuplicateReferencia
		case "payments_idem_key_key":
			return ErrIdempotencyConflict
		}
	}
	return err
}

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
