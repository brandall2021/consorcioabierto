// Package cuenta_corriente modela el libro de movimientos por UF: cargos
// (charges) y asientos de cuenta corriente (account_entries). H3.5 (§5.3).
// Las funciones reciben un *db.Queries ya ligado a la transacción del handler
// (RLS de tenant activo); aquí no se abren transacciones ni se controla el
// commit. La cobertura con base de datos llega con los tests de integración
// de H3.7 (Task 12).
package cuenta_corriente

import (
	"context"
	"errors"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrChargeNotFound = errors.New("cargo no encontrado")
	ErrCargoInvalido  = errors.New("datos de cargo inválidos")
)

type ChargeDTO struct {
	ID            string  `json:"id"`
	UnidadID      string  `json:"unidad_id"`
	LiquidacionID *string `json:"liquidacion_id"`
	Concepto      string  `json:"concepto"`
	DueDate       string  `json:"due_date"`
	TotalCents    int64   `json:"total_cents"`
	SaldoCents    int64   `json:"saldo_cents"`
}

type AccountEntryDTO struct {
	ID            string  `json:"id"`
	UnidadID      string  `json:"unidad_id"`
	Tipo          string  `json:"tipo"`
	FechaEfectiva string  `json:"fecha_efectiva"`
	DebitCents    int64   `json:"debit_cents"`
	CreditCents   int64   `json:"credit_cents"`
	Currency      string  `json:"currency"`
	Referencia    *string `json:"referencia"`
}

type UnidadCobro struct {
	UnidadID      string
	Concepto      string
	DueDate       string
	TotalCents    int64
	LiquidacionID *string
}

// parseDueDate convierte "AAAA-MM-DD" a pgtype.Date. Fallo = datos inválidos
// (R4: no insertar fecha cero en silencio).
func parseDueDate(s string) (pgtype.Date, error) {
	var d pgtype.Date
	if err := d.Scan(s); err != nil {
		return pgtype.Date{}, err
	}
	return d, nil
}

// parseUUID convierte un id a pgtype.UUID. Fallo = dato inválido del caller.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}

// CreateCargos genera un charge y su asiento 'cargo' para cada unidad del
// lote. Charge.saldo = total y entry DebitCents = total / CreditCents = 0.
// Ambos se insertan en la misma transacción externa (la confirma el handler).
func CreateCargos(ctx context.Context, q *db.Queries, unidades []UnidadCobro, currency string) ([]ChargeDTO, []AccountEntryDTO, error) {
	charges := make([]ChargeDTO, 0, len(unidades))
	entries := make([]AccountEntryDTO, 0, len(unidades))

	for _, u := range unidades {
		uid, err := parseUUID(u.UnidadID)
		if err != nil {
			return nil, nil, ErrCargoInvalido
		}
		lid, err := parseLiquidacionID(u.LiquidacionID)
		if err != nil {
			return nil, nil, ErrCargoInvalido
		}
		due, err := parseDueDate(u.DueDate)
		if err != nil {
			return nil, nil, ErrCargoInvalido
		}

		ch, err := q.InsertCharge(ctx, db.InsertChargeParams{
			UnidadID:      uid,
			LiquidacionID: lid,
			Concepto:      u.Concepto,
			DueDate:       due,
			TotalCents:    u.TotalCents,
			SaldoCents:    u.TotalCents,
		})
		if err != nil {
			return nil, nil, err
		}

		charges = append(charges, chargeDTO(ch))

		entry, err := q.InsertAccountEntry(ctx, db.InsertAccountEntryParams{
			UnidadID:      uid,
			Tipo:          "cargo",
			FechaEfectiva: due,
			DebitCents:    u.TotalCents,
			CreditCents:   0,
			Currency:      currency,
			Referencia:    pgtype.Text{},
			ReversaDeID:   pgtype.UUID{},
			ChargeID:      ch.ID,
		})
		if err != nil {
			return nil, nil, err
		}

		entries = append(entries, accountEntryDTO(entry))
	}

	return charges, entries, nil
}

// RevertirCargo crea una reversa para un charge existente: asiento 'reversa'
// con CreditCents = total del cargo y saldo del charge en 0. La referencia
// registra el cargo original.
func RevertirCargo(ctx context.Context, q *db.Queries, chargeID, unidadID, currency string) (AccountEntryDTO, error) {
	cid, err := parseUUID(chargeID)
	if err != nil {
		return AccountEntryDTO{}, ErrChargeNotFound
	}

	ch, err := q.GetCharge(ctx, cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountEntryDTO{}, ErrChargeNotFound
	}
	if err != nil {
		return AccountEntryDTO{}, err
	}

	// El charge ya está verificado como del tenant activo (RLS), así que si el
	// unidad_id del caller no parsea, se conserva el de la UF del cargo.
	uid, err := parseUUID(unidadID)
	if err != nil {
		uid = ch.UnidadID
	}

	if err := q.UpdateChargeSaldo(ctx, db.UpdateChargeSaldoParams{
		ID:         ch.ID,
		SaldoCents: 0,
	}); err != nil {
		return AccountEntryDTO{}, err
	}

	referencia := "reversa de cargo " + chargeID
	entry, err := q.InsertAccountEntry(ctx, db.InsertAccountEntryParams{
		UnidadID:      uid,
		Tipo:          "reversa",
		FechaEfectiva: pgtype.Date{Time: ch.DueDate.Time, Valid: true},
		DebitCents:    0,
		CreditCents:   ch.TotalCents,
		Currency:      currency,
		Referencia:    pgtype.Text{String: referencia, Valid: true},
		ReversaDeID:   pgtype.UUID{},
		ChargeID:      ch.ID,
	})
	if err != nil {
		return AccountEntryDTO{}, err
	}

	return accountEntryDTO(entry), nil
}

func parseLiquidacionID(l *string) (pgtype.UUID, error) {
	if l == nil {
		return pgtype.UUID{}, nil
	}
	id, err := parseUUID(*l)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

func chargeDTO(c db.Charge) ChargeDTO {
	out := ChargeDTO{
		ID:         c.ID.String(),
		UnidadID:   c.UnidadID.String(),
		Concepto:   c.Concepto,
		DueDate:    c.DueDate.Time.Format("2006-01-02"),
		TotalCents: c.TotalCents,
		SaldoCents: c.SaldoCents,
	}
	if c.LiquidacionID.Valid {
		s := c.LiquidacionID.String()
		out.LiquidacionID = &s
	}
	return out
}

func accountEntryDTO(e db.AccountEntry) AccountEntryDTO {
	out := AccountEntryDTO{
		ID:            e.ID.String(),
		UnidadID:      e.UnidadID.String(),
		Tipo:          e.Tipo,
		FechaEfectiva: e.FechaEfectiva.Time.Format("2006-01-02"),
		DebitCents:    e.DebitCents,
		CreditCents:   e.CreditCents,
		Currency:      e.Currency,
	}
	if e.Referencia.Valid {
		s := e.Referencia.String
		out.Referencia = &s
	}
	return out
}