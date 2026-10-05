// Package personas administra el vinculo entre una persona y el usuario de la
// plataforma que la representa.
//
// El vinculo vive en personas.user_id (migracion 00027) y es el eslabon que
// permite que el portal del consorcista se acote por vinculo vigente:
// user -> personas -> unidad_personas -> unidades. Sin el, el portal solo
// podia acotar por tenant y cualquier consorcista veia las unidades ajenas.
//
// Reglas:
//   - El tenant lo aporta el RLS (app.current_tenant_id()); el cliente nunca
//     elige el tenant ni el consorcio.
//   - Solo se puede vincular un usuario que sea miembro del tenant activo.
//   - Un usuario no puede quedar vinculado a dos personas del mismo tenant
//     (indice unico parcial en 00027).
package personas

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
)

var (
	// ErrPersonaNotFound no distingue "de otro tenant" de inexistente: la
	// operacion responde 404 en ambos casos para no filtrar existencia ajena.
	ErrPersonaNotFound = errors.New("persona no encontrada")
	// ErrUsuarioNotFound es el mismo criterio para el usuario a vincular.
	ErrUsuarioNotFound = errors.New("usuario no encontrado")
	// ErrVinculoInvalido cubre entradas mal formadas (UUID o cuerpo vacío).
	ErrVinculoInvalido = errors.New("vínculo inválido")
	// ErrUsuarioYaVinculado evita que un usuario represente a dos personas.
	ErrUsuarioYaVinculado = errors.New("el usuario ya está vinculado a otra persona")
)

// Queryer agrupa las consultas que la operacion necesita. Permite probar el
// dominio con dobles sin levantar Postgres.
type Queryer interface {
	ListTenantMembers(context.Context) ([]db.ListTenantMembersRow, error)
	GetPersonaForVinculo(context.Context, pgtype.UUID) (db.PersonaVinculable, error)
	SetPersonaUsuario(context.Context, db.SetPersonaUsuarioParams) error
	UnsetPersonaUsuario(context.Context, pgtype.UUID) error
}

// MiembroDTO es un miembro del tenant activo (contrato HTTP).
type MiembroDTO struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Nombre    string `json:"nombre"`
	Membresia string `json:"membresia"`
	Estado    string `json:"estado"`
	Vinculado *bool  `json:"vinculado"`
}

// PersonaVinculoDTO es la persona con su usuario asociado (contrato HTTP).
type PersonaVinculoDTO struct {
	ID        string  `json:"id"`
	Nombre    string  `json:"nombre"`
	Documento *string `json:"documento"`
	Email     *string `json:"email"`
	Telefono  *string `json:"telefono"`
	UserID    *string `json:"user_id"`
}

// ListarMiembros devuelve los miembros del tenant activo que se pueden
// vincular a una persona. Los usuarios ya vinculados se marcan, para que la UI
// no ofrezca crear una ambigüedad.
func ListarMiembros(ctx context.Context, q Queryer) ([]MiembroDTO, error) {
	rows, err := q.ListTenantMembers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MiembroDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, MiembroDTO{
			UserID:    r.UserID.String(),
			Email:     r.EmailNormalized,
			Nombre:    r.Name,
			Membresia: r.MembershipStatus,
			Estado:    r.UserStatus,
		})
	}
	return out, nil
}

// Vincular asocia una persona del tenant activo con un usuario del tenant.
// Revincular la misma persona con el mismo usuario es idempotente; cambiar a
// otro usuario esta permitido y queda registrado en auditoria por el handler.
func Vincular(ctx context.Context, q Queryer, personaID, usuarioID string) (PersonaVinculoDTO, error) {
	persona, err := uuidToPG(personaID)
	if err != nil {
		return PersonaVinculoDTO{}, err
	}
	usuario, err := uuidToPG(usuarioID)
	if err != nil {
		return PersonaVinculoDTO{}, err
	}
	if strings.TrimSpace(personaID) == "" || strings.TrimSpace(usuarioID) == "" {
		return PersonaVinculoDTO{}, ErrVinculoInvalido
	}

	// La persona tiene que pertenecer al tenant activo. La consulta ya filtra
	// por tenant, asi que una persona de otro tenant es indistinguible de una
	// inexistente.
	actual, err := q.GetPersonaForVinculo(ctx, persona)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PersonaVinculoDTO{}, ErrPersonaNotFound
		}
		return PersonaVinculoDTO{}, err
	}

	// El usuario tiene que ser miembro del tenant activo. Sin esta comprobacion
	// se podria colgar una persona de un usuario de otro tenant.
	members, err := q.ListTenantMembers(ctx)
	if err != nil {
		return PersonaVinculoDTO{}, err
	}
	if !esMiembro(members, usuario) {
		return PersonaVinculoDTO{}, ErrUsuarioNotFound
	}

	if err := q.SetPersonaUsuario(ctx, db.SetPersonaUsuarioParams{
		ID:     persona,
		UserID: usuario,
	}); err != nil {
		if isUniqueViolation(err) {
			return PersonaVinculoDTO{}, ErrUsuarioYaVinculado
		}
		return PersonaVinculoDTO{}, err
	}

	actual.UserID = usuario
	return toDTO(actual), nil
}

// Desvincular borra la asociacion sin tocar la persona ni sus vinculos con UFs.
func Desvincular(ctx context.Context, q Queryer, personaID string) (PersonaVinculoDTO, error) {
	persona, err := uuidToPG(personaID)
	if err != nil {
		return PersonaVinculoDTO{}, err
	}
	actual, err := q.GetPersonaForVinculo(ctx, persona)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PersonaVinculoDTO{}, ErrPersonaNotFound
		}
		return PersonaVinculoDTO{}, err
	}
	if !actual.UserID.Valid {
		return toDTO(actual), nil
	}
	if err := q.UnsetPersonaUsuario(ctx, persona); err != nil {
		return PersonaVinculoDTO{}, err
	}
	actual.UserID = pgtype.UUID{}
	return toDTO(actual), nil
}

func esMiembro(rows []db.ListTenantMembersRow, usuario pgtype.UUID) bool {
	for _, r := range rows {
		if r.UserID == usuario {
			return true
		}
	}
	return false
}

func toDTO(p db.PersonaVinculable) PersonaVinculoDTO {
	dto := PersonaVinculoDTO{ID: p.ID.String(), Nombre: p.Nombre}
	if p.Documento.Valid {
		s := p.Documento.String
		dto.Documento = &s
	}
	if p.Email.Valid {
		s := p.Email.String
		dto.Email = &s
	}
	if p.Telefono.Valid {
		s := p.Telefono.String
		dto.Telefono = &s
	}
	if p.UserID.Valid {
		s := p.UserID.String()
		dto.UserID = &s
	}
	return dto
}

func uuidToPG(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: %s", ErrVinculoInvalido, s)
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
