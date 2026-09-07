package expensas

import (
	"context"
	"errors"
	"regexp"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrLiquidacionNotFound           = errors.New("liquidación no encontrada")
	ErrLiquidacionInvalid            = errors.New("datos de liquidación inválidos")
	ErrLiquidacionTransicionInvalida = errors.New("transición de estado no permitida")
	ErrLiquidacionConflict           = errors.New("ya existe una liquidación activa para ese período")
	ErrLiquidacionVersionMismatch    = errors.New("versión no coincide (conflicto concurrente)")
)

var validTransitions = map[string]map[string]bool{
	"borrador":   {"calculada": true, "anulada": true},
	"calculada":  {"borrador": true, "confirmada": true, "anulada": true},
	"confirmada": {"publicada": true, "anulada": true},
	"publicada":  {"cerrada": true},
}

func canTransition(from, to string) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

var periodoRe = regexp.MustCompile(`^[0-9]{6}$`)

func validatePeriodo(p string) error {
	if !periodoRe.MatchString(p) {
		return ErrLiquidacionInvalid
	}
	return nil
}

func ValidateVencimientos(v1 string, v2 *string) error {
	t1, err := time.Parse("2006-01-02", v1)
	if err != nil {
		return ErrLiquidacionInvalid
	}
	if v2 != nil {
		t2, err := time.Parse("2006-01-02", *v2)
		if err != nil {
			return ErrLiquidacionInvalid
		}
		if t2.Before(t1) {
			return ErrLiquidacionInvalid
		}
	}
	return nil
}

type LiquidacionDTO struct {
	ID                    string    `json:"id"`
	ConsorcioID           string    `json:"consorcio_id"`
	Periodo               string    `json:"periodo"`
	Vencimiento1          string    `json:"vencimiento_1"`
	Vencimiento2          *string   `json:"vencimiento_2"`
	Estado                string    `json:"estado"`
	Version               int       `json:"version"`
	TotalGastosCents      int64     `json:"total_gastos_cents"`
	TotalDistribuidoCents int64     `json:"total_distribuido_cents"`
	UnidadesAlcanzadas    int       `json:"unidades_alcanzadas"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type LiquidacionFilter struct {
	Periodo string
	Estado  string
}

func liquidacionDTO(r db.Liquidacion) LiquidacionDTO {
	dto := LiquidacionDTO{
		ID:                    r.ID.String(),
		ConsorcioID:           r.ConsorcioID.String(),
		Periodo:               r.Periodo,
		Vencimiento1:          r.Vencimiento1.Time.Format("2006-01-02"),
		Estado:                r.Estado,
		Version:               int(r.Version),
		TotalGastosCents:      r.TotalGastosCents,
		TotalDistribuidoCents: r.TotalDistribuidoCents,
		UnidadesAlcanzadas:    int(r.UnidadesAlcanzadas),
		CreatedAt:             r.CreatedAt.Time,
		UpdatedAt:             r.UpdatedAt.Time,
	}
	if r.Vencimiento2.Valid {
		s := r.Vencimiento2.Time.Format("2006-01-02")
		dto.Vencimiento2 = &s
	}
	return dto
}

func CreateLiquidacion(ctx context.Context, q *db.Queries, consorcioID, periodo, v1 string, v2 *string) (LiquidacionDTO, error) {
	if err := validatePeriodo(periodo); err != nil {
		return LiquidacionDTO{}, err
	}
	if err := ValidateVencimientos(v1, v2); err != nil {
		return LiquidacionDTO{}, err
	}
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionInvalid
	}
	var v1Date pgtype.Date
	if err := v1Date.Scan(v1); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionInvalid
	}
	var v2Date pgtype.Date
	if v2 != nil {
		if err := v2Date.Scan(*v2); err != nil {
			return LiquidacionDTO{}, ErrLiquidacionInvalid
		}
	}
	row, err := q.CreateLiquidacion(ctx, db.CreateLiquidacionParams{
		ConsorcioID:  cid,
		Periodo:      periodo,
		Vencimiento1: v1Date,
		Vencimiento2: v2Date,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LiquidacionDTO{}, ErrLiquidacionConflict
		}
		return LiquidacionDTO{}, err
	}
	return liquidacionDTO(row), nil
}

func GetLiquidacion(ctx context.Context, q *db.Queries, consorcioID, id string) (LiquidacionDTO, error) {
	var cid, lid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionInvalid
	}
	if err := lid.Scan(id); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionNotFound
	}
	row, err := q.GetLiquidacion(ctx, db.GetLiquidacionParams{ConsorcioID: cid, ID: lid})
	if errors.Is(err, pgx.ErrNoRows) {
		return LiquidacionDTO{}, ErrLiquidacionNotFound
	}
	if err != nil {
		return LiquidacionDTO{}, err
	}
	return liquidacionDTO(row), nil
}

func ListLiquidaciones(ctx context.Context, q *db.Queries, consorcioID string, f LiquidacionFilter) ([]LiquidacionDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return nil, ErrLiquidacionInvalid
	}
	rows, err := q.ListLiquidaciones(ctx, db.ListLiquidacionesParams{
		ConsorcioID: cid,
		Periodo:     f.Periodo,
		Estado:      f.Estado,
	})
	if err != nil {
		return nil, err
	}
	out := make([]LiquidacionDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, liquidacionDTO(r))
	}
	return out, nil
}

func UpdateVencimientos(ctx context.Context, q *db.Queries, consorcioID, id, v1 string, v2 *string, expectedVersion int) error {
	if err := ValidateVencimientos(v1, v2); err != nil {
		return err
	}
	var cid, lid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return ErrLiquidacionInvalid
	}
	if err := lid.Scan(id); err != nil {
		return ErrLiquidacionNotFound
	}
	var v1Date pgtype.Date
	if err := v1Date.Scan(v1); err != nil {
		return ErrLiquidacionInvalid
	}
	var v2Date pgtype.Date
	if v2 != nil {
		if err := v2Date.Scan(*v2); err != nil {
			return ErrLiquidacionInvalid
		}
	}
	err := q.UpdateLiquidacionVencimientos(ctx, db.UpdateLiquidacionVencimientosParams{
		ConsorcioID:     cid,
		ID:              lid,
		Vencimiento1:    v1Date,
		Vencimiento2:    v2Date,
		ExpectedVersion: int32(expectedVersion),
	})
	if err != nil {
		return err
	}
	return nil
}

func TransitionEstado(ctx context.Context, q *db.Queries, consorcioID, id, current, next string, expectedVersion int) error {
	if !canTransition(current, next) {
		return ErrLiquidacionTransicionInvalida
	}
	var cid, lid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return ErrLiquidacionInvalid
	}
	if err := lid.Scan(id); err != nil {
		return ErrLiquidacionNotFound
	}
	n, err := q.TransitionLiquidacion(ctx, db.TransitionLiquidacionParams{
		ConsorcioID:     cid,
		ID:              lid,
		EstadoActual:    current,
		NuevoEstado:     next,
		ExpectedVersion: int32(expectedVersion),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLiquidacionVersionMismatch
	}
	return nil
}
