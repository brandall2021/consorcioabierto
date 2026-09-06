// Conceptos de expensa (H3.1, §5.1). consorcio_id NULL = concepto de tenant
// (para todos sus consorcios). regla 'coeficiente' es la única soportada.
package expensas

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrConceptoInvalid      = errors.New("concepto inválido")
	ErrConceptoNotFound     = errors.New("concepto no encontrado")
	ErrConceptoDuplicate    = errors.New("ya existe un concepto con ese nombre")
	ErrConceptoReglaInvalid = errors.New("regla de distribución no soportada")
)

var validConceptoReglas = map[string]bool{"coeficiente": true}
var _ = validConceptoReglas
var validConceptoCategorias = map[string]bool{"administracion": true, "servicios": true, "mantenimiento": true, "fondos": true, "otros": true}

type ConceptoDTO struct {
	ID          string    `json:"id"`
	ConsorcioID *string   `json:"consorcio_id"`
	Nombre      string    `json:"nombre"`
	Categoria   string    `json:"categoria"`
	Regla       string    `json:"regla"`
	CreatedAt   time.Time `json:"created_at"`
}

type ConceptoFilter struct {
	Q string
}

type ConceptoInput struct {
	Nombre    string `json:"nombre"`
	Categoria string `json:"categoria"`
}

// ListConceptos devuelve conceptos del consorcio y, además, los de tenant
// (consorcio_id NULL), los dos del tenant activo.
func ListConceptos(ctx context.Context, q *db.Queries, consorcioID string, f ConceptoFilter) ([]ConceptoDTO, error) {
	var cid pgtype.UUID
	hasCID := consorcioID != ""
	if hasCID {
		if err := cid.Scan(consorcioID); err != nil {
			return nil, ErrConceptoInvalid
		}
	}
	rows, err := q.ListConceptos(ctx, db.ListConceptosParams{
		ConsorcioID: cid,
		Q:           strings.TrimSpace(f.Q),
	})
	if err != nil {
		return nil, err
	}
	out := make([]ConceptoDTO, 0, len(rows))
	for _, c := range rows {
		out = append(out, conceptoDTO(c))
	}
	return out, nil
}

// CreateConcepto crea un concepto de consorcio (consorcioID obligatorio).
func CreateConcepto(ctx context.Context, q *db.Queries, consorcioID string, in ConceptoInput) (ConceptoDTO, error) {
	v, err := validateConcepto(in)
	if err != nil {
		return ConceptoDTO{}, err
	}
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return ConceptoDTO{}, ErrConceptoInvalid
	}
	created, err := q.CreateConcepto(ctx, db.CreateConceptoParams{
		ConsorcioID: cid,
		Nombre:      v.nombre,
		Categoria:   v.categoria,
	})
	if err != nil {
		return ConceptoDTO{}, mapPGError(err, ErrConceptoDuplicate)
	}
	return conceptoDTO(created), nil
}

// GetConcepto devuelve un concepto del tenant activo por id.
func GetConcepto(ctx context.Context, q *db.Queries, conceptoID string) (ConceptoDTO, error) {
	var id pgtype.UUID
	if err := id.Scan(conceptoID); err != nil {
		return ConceptoDTO{}, ErrConceptoNotFound
	}
	c, err := q.GetConcepto(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConceptoDTO{}, ErrConceptoNotFound
	}
	if err != nil {
		return ConceptoDTO{}, err
	}
	return conceptoDTO(c), nil
}

type validatedConcepto struct {
	nombre    string
	categoria string
}

func validateConcepto(in ConceptoInput) (validatedConcepto, error) {
	nombre := strings.TrimSpace(in.Nombre)
	categoria := strings.TrimSpace(in.Categoria)
	if nombre == "" || len(nombre) > 100 {
		return validatedConcepto{}, ErrConceptoInvalid
	}
	if !validConceptoCategorias[categoria] {
		return validatedConcepto{}, ErrConceptoInvalid
	}
	return validatedConcepto{nombre: nombre, categoria: categoria}, nil
}

// mapPGError traduce el código 23505 (unique_violation) al error de dominio.
func mapPGError(err error, dup error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return dup
	}
	return err
}

func conceptoDTO(c db.ConceptosExpensa) ConceptoDTO {
	out := ConceptoDTO{
		ID:        c.ID.String(),
		Nombre:    c.Nombre,
		Categoria: c.Categoria,
		Regla:     c.Regla,
		CreatedAt: c.CreatedAt.Time,
	}
	if c.ConsorcioID.Valid && c.ConsorcioID.Bytes != [16]byte{} {
		s := c.ConsorcioID.String()
		out.ConsorcioID = &s
	}
	return out
}
