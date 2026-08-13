// Package consorcios: CRUD tenant-scoped de consorcios ([ADR-0009], §5.4).
// Las funciones reciben un *db.Queries ligado a una transacción con el
// contexto de RLS ya seteado (tenancy.SetContext), igual que internal/audit.
package consorcios

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// ErrNotFound: el consorcio no existe en el tenant activo (o no es visible).
	ErrNotFound = errors.New("consorcio no encontrado")
	// ErrInvalid: payload inválido (nombre/cuit/tipo/estado).
	ErrInvalid = errors.New("consorcio inválido")
	// ErrDuplicateName: ya existe un consorcio con el mismo nombre en el tenant.
	ErrDuplicateName = errors.New("ya existe un consorcio con ese nombre")
)

var cuitRe = regexp.MustCompile(`^[0-9]{11}$`)

var validTipos = map[string]bool{"edificio": true, "barrio": true, "complejo": true, "otros": true}
var validEstados = map[string]bool{"activo": true, "inactivo": true}

// DTO es la representación pública de un consorcio (contrato HTTP).
type DTO struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Nombre    string    `json:"nombre"`
	Cuit      *string   `json:"cuit"`
	Domicilio *string   `json:"domicilio"`
	Tipo      string    `json:"tipo"`
	Estado    string    `json:"estado"`
	CreatedAt time.Time `json:"created_at"`
}

// Filter acota el listado de consorcios.
type Filter struct {
	Q      string
	Estado string
}

// Input es el payload de creación/actualización (PATCH parcial).
type Input struct {
	Nombre    *string `json:"nombre"`
	Cuit      *string `json:"cuit"`
	Domicilio *string `json:"domicilio"`
	Tipo      *string `json:"tipo"`
	Estado    *string `json:"estado"`
}

// Create valida y crea un consorcio en el tenant activo.
func Create(ctx context.Context, q *db.Queries, in Input) (DTO, error) {
	validated, err := validate(in, true)
	if err != nil {
		return DTO{}, err
	}

	c, err := q.CreateConsorcio(ctx, db.CreateConsorcioParams{
		Nombre:    validated.nombre,
		Cuit:      textOrNil(validated.cuit),
		Domicilio: textOrNil(validated.domicilio),
		Tipo:      validated.tipo,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return DTO{}, ErrDuplicateName
		}
		return DTO{}, err
	}
	return toDTO(c), nil
}

// Get devuelve un consorcio del tenant activo.
func Get(ctx context.Context, q *db.Queries, id string) (DTO, error) {
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return DTO{}, ErrInvalid
	}
	c, err := q.GetConsorcio(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return DTO{}, ErrNotFound
	}
	if err != nil {
		return DTO{}, err
	}
	return toDTO(c), nil
}

// List devuelve consorcios del tenant activo con filtros.
func List(ctx context.Context, q *db.Queries, f Filter) ([]DTO, error) {
	rows, err := q.ListConsorcios(ctx, db.ListConsorciosParams{Q: f.Q, Estado: f.Estado})
	if err != nil {
		return nil, err
	}
	out := make([]DTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDTO(r))
	}
	return out, nil
}

// Update aplica un PATCH parcial sobre un consorcio del tenant activo.
func Update(ctx context.Context, q *db.Queries, id string, in Input) (DTO, error) {
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return DTO{}, ErrInvalid
	}
	params := db.UpdateConsorcioParams{ID: uid}
	if in.Nombre != nil {
		if err := checkNombre(*in.Nombre); err != nil {
			return DTO{}, err
		}
		params.Nombre = pgtype.Text{String: strings.TrimSpace(*in.Nombre), Valid: true}
	}
	if in.Cuit != nil {
		if !cuitRe.MatchString(*in.Cuit) {
			return DTO{}, ErrInvalid
		}
		params.Cuit = pgtype.Text{String: *in.Cuit, Valid: true}
	}
	if in.Domicilio != nil {
		if len(*in.Domicilio) > 300 {
			return DTO{}, ErrInvalid
		}
		params.Domicilio = pgtype.Text{String: *in.Domicilio, Valid: true}
	}
	if in.Tipo != nil {
		if !validTipos[*in.Tipo] {
			return DTO{}, ErrInvalid
		}
		params.Tipo = pgtype.Text{String: *in.Tipo, Valid: true}
	}
	if in.Estado != nil {
		if !validEstados[*in.Estado] {
			return DTO{}, ErrInvalid
		}
		params.Estado = pgtype.Text{String: *in.Estado, Valid: true}
	}

	c, err := q.UpdateConsorcio(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return DTO{}, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return DTO{}, ErrDuplicateName
		}
		return DTO{}, err
	}
	return toDTO(c), nil
}

type validated struct {
	nombre    string
	cuit      string
	domicilio string
	tipo      string
}

func validate(in Input, requireNombre bool) (validated, error) {
	var out validated
	if requireNombre {
		if in.Nombre == nil || strings.TrimSpace(*in.Nombre) == "" {
			return out, ErrInvalid
		}
	}
	if in.Nombre != nil {
		if err := checkNombre(*in.Nombre); err != nil {
			return out, err
		}
		out.nombre = strings.TrimSpace(*in.Nombre)
	}
	if in.Cuit != nil {
		if !cuitRe.MatchString(*in.Cuit) {
			return out, ErrInvalid
		}
		out.cuit = *in.Cuit
	}
	if in.Domicilio != nil {
		if len(*in.Domicilio) > 300 {
			return out, ErrInvalid
		}
		out.domicilio = *in.Domicilio
	}
	if in.Tipo != nil {
		if !validTipos[*in.Tipo] {
			return out, ErrInvalid
		}
		out.tipo = *in.Tipo
	} else if requireNombre {
		out.tipo = "edificio"
	}
	return out, nil
}

func checkNombre(n string) error {
	n = strings.TrimSpace(n)
	if n == "" || len(n) > 200 {
		return ErrInvalid
	}
	return nil
}

func toDTO(c db.Consorcio) DTO {
	dto := DTO{
		ID:        c.ID.String(),
		TenantID:  c.TenantID.String(),
		Nombre:    c.Nombre,
		Tipo:      c.Tipo,
		Estado:    c.Estado,
		CreatedAt: c.CreatedAt.Time,
	}
	if c.Cuit.Valid {
		s := c.Cuit.String
		dto.Cuit = &s
	}
	if c.Domicilio.Valid {
		s := c.Domicilio.String
		dto.Domicilio = &s
	}
	return dto
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func textOrNil(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
