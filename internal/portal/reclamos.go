package portal

import (
	"context"
	"errors"
	"strings"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/reclamos"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrSinVinculo indica que el usuario no tiene vinculo vigente con la unidad.
// No se distingue de una unidad inexistente para no filtrar su existencia.
var ErrSinVinculo = errors.New("no tenés un vínculo vigente con esa unidad")

// UnidadesDelUsuario es la parte de Queryer que resuelve el scope del portal.
type UnidadesDelUsuario interface {
	ListUnidadesForCurrentUser(context.Context) ([]db.Unidade, error)
}

// CrearReclamo da de alta un reclamo desde el portal del consorcista.
//
// El alcance lo decide el servidor: la unidad debe estar entre las del usuario
// con vinculo vigente y el consorcio se deriva de esa unidad. El cliente nunca
// elige el consorcio, asi que no puede abrir un reclamo en otro tenant ni en una
// unidad ajena. Los permisos no cambian: alcanza con ser usuario autenticado
// del tenant, porque el vinculo es lo que acota (ver [ADR-0009]).
func CrearReclamo(ctx context.Context, unidades UnidadesDelUsuario, q reclamos.Queryer, in reclamos.CreateInput) (reclamos.Reclamo, error) {
	unidadID := strings.TrimSpace(in.UnidadID)
	if unidadID == "" {
		return reclamos.Reclamo{}, ErrSinVinculo
	}
	id, err := parseUUID(unidadID)
	if err != nil {
		return reclamos.Reclamo{}, ErrSinVinculo
	}

	mias, err := unidades.ListUnidadesForCurrentUser(ctx)
	if err != nil {
		return reclamos.Reclamo{}, err
	}
	var authorize bool
	var consorcioID pgtype.UUID
	for _, u := range mias {
		if u.ID == id {
			authorize = true
			consorcioID = u.ConsorcioID
			break
		}
	}
	if !authorize {
		return reclamos.Reclamo{}, ErrSinVinculo
	}

	return reclamos.Create(ctx, q, consorcioID.String(), in)
}

func parseUUID(s string) (pgtype.UUID, error) {
	var out pgtype.UUID
	if err := out.Scan(s); err != nil {
		return out, err
	}
	if !out.Valid {
		return out, errors.New("id inválido")
	}
	return out, nil
}
