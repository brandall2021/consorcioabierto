// Gastos con comprobante y documento opcionales (H3.1, §5.1).
package expensas

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrGastoInvalid  = errors.New("gasto inválido")
	ErrGastoNotFound = errors.New("gasto no encontrado")
)

var validGastoEstados = map[string]bool{"registrado": true, "pagado": true, "anulado": true}
var _ = validGastoEstados

type GastoDTO struct {
	ID             string    `json:"id"`
	ConsorcioID    string    `json:"consorcio_id"`
	ConceptoID     string    `json:"concepto_id"`
	ConceptoNombre string    `json:"concepto_nombre"`
	ProveedorID    *string   `json:"proveedor_id"`
	Comprobante    *string   `json:"comprobante"`
	ImporteCents   int64     `json:"importe_cents"`
	Fecha          string    `json:"fecha"`
	Estado         string    `json:"estado"`
	DocumentoID    *string   `json:"documento_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type GastoFilter struct {
	Mes         string
	ProveedorID string
}

type GastoInput struct {
	ConceptoID   string  `json:"concepto_id"`
	ProveedorID  *string `json:"proveedor_id"`
	Comprobante  *string `json:"comprobante"`
	ImporteCents int64   `json:"importe_cents"`
	Fecha        string  `json:"fecha"`
	DocumentoID  *string `json:"documento_id"`
}

func ListGastos(ctx context.Context, q *db.Queries, consorcioID string, f GastoFilter) ([]GastoDTO, error) {
	var cid, pid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return nil, ErrGastoInvalid
	}
	if f.ProveedorID != "" {
		if err := pid.Scan(f.ProveedorID); err != nil {
			return nil, ErrGastoInvalid
		}
	}
	rows, err := q.ListGastos(ctx, db.ListGastosParams{
		ConsorcioID: cid,
		Mes:         strings.TrimSpace(f.Mes),
		ProveedorID: pid,
	})
	if err != nil {
		return nil, err
	}
	out := make([]GastoDTO, 0, len(rows))
	for _, g := range rows {
		out = append(out, gastoDTO(g))
	}
	return out, nil
}

func CreateGasto(ctx context.Context, q *db.Queries, consorcioID string, in GastoInput) (GastoDTO, error) {
	v, err := validateGasto(in)
	if err != nil {
		return GastoDTO{}, err
	}
	var cid, conceptoID, proveedorID, documentoID pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return GastoDTO{}, ErrGastoInvalid
	}
	if err := conceptoID.Scan(in.ConceptoID); err != nil {
		return GastoDTO{}, ErrGastoInvalid
	}
	if in.ProveedorID != nil {
		if err := proveedorID.Scan(*in.ProveedorID); err != nil {
			return GastoDTO{}, ErrGastoInvalid
		}
	}
	if in.DocumentoID != nil {
		if err := documentoID.Scan(*in.DocumentoID); err != nil {
			return GastoDTO{}, ErrGastoInvalid
		}
	}

	created, err := q.CreateGasto(ctx, db.CreateGastoParams{
		ConsorcioID:  cid,
		ConceptoID:   conceptoID,
		ProveedorID:  proveedorID,
		Comprobante:  textOrNil(in.Comprobante),
		ImporteCents: v.importeCents,
		Fecha:        v.fecha,
		DocumentoID:  documentoID,
	})
	if err != nil {
		return GastoDTO{}, err
	}
	return gastoDTO(created), nil
}

func GetGasto(ctx context.Context, q *db.Queries, consorcioID, gastoID string) (GastoDTO, error) {
	var cid, gid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return GastoDTO{}, ErrGastoInvalid
	}
	if err := gid.Scan(gastoID); err != nil {
		return GastoDTO{}, ErrGastoNotFound
	}
	g, err := q.GetGasto(ctx, db.GetGastoParams{ConsorcioID: cid, ID: gid})
	if errors.Is(err, pgx.ErrNoRows) {
		return GastoDTO{}, ErrGastoNotFound
	}
	if err != nil {
		return GastoDTO{}, err
	}
	return gastoDTO(g), nil
}

type validatedGasto struct {
	importeCents int64
	fecha        pgtype.Date
}

// fechaEnFormato acepta ISO YYYY-MM-DD. No permite fechas futuras (hoy o antes).
func validateGasto(in GastoInput) (validatedGasto, error) {
	if in.ImporteCents <= 0 {
		return validatedGasto{}, ErrGastoInvalid
	}
	f, err := time.Parse("2006-01-02", strings.TrimSpace(in.Fecha))
	if err != nil {
		return validatedGasto{}, ErrGastoInvalid
	}
	if f.After(time.Now()) {
		return validatedGasto{}, ErrGastoInvalid
	}
	var pd pgtype.Date
	if err := pd.Scan(f.Format("2006-01-02")); err != nil {
		return validatedGasto{}, ErrGastoInvalid
	}
	return validatedGasto{importeCents: in.ImporteCents, fecha: pd}, nil
}

func textOrNil(s *string) pgtype.Text {
	var t pgtype.Text
	if s == nil || strings.TrimSpace(*s) == "" {
		return t
	}
	t.Valid = true
	t.String = strings.TrimSpace(*s)
	return t
}

func gastoDTO(g db.Gasto) GastoDTO {
	out := GastoDTO{
		ID:           g.ID.String(),
		ConsorcioID:  g.ConsorcioID.String(),
		ConceptoID:   g.ConceptoID.String(),
		ProveedorID:  uuidPtrOrNil(g.ProveedorID),
		Comprobante:  textPtrOrNil(g.Comprobante),
		ImporteCents: g.ImporteCents,
		Fecha:        g.Fecha.Time.Format("2006-01-02"),
		Estado:       g.Estado,
		DocumentoID:  uuidPtrOrNil(g.DocumentoID),
		CreatedAt:    g.CreatedAt.Time,
	}
	return out
}

func uuidPtrOrNil(u pgtype.UUID) *string {
	if !u.Valid || u.Bytes == [16]byte{} {
		return nil
	}
	s := u.String()
	return &s
}

func textPtrOrNil(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}
