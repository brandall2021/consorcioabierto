package consorcios

import (
	"context"
	"errors"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// UFs, personas y vínculos históricos ([ADRs], §5.1, H2.2).
// Las funciones reciben un *db.Queries ligado a una transacción con el contexto
// de RLS seteado, igual que consorcios. Los vínculos nunca se sobreescriben:
// se cierra valid_to del vigente y se crea otro registro.

var (
	// ErrUnidadNotFound: la UF no existe en el consorcio/tenant activo.
	ErrUnidadNotFound = errors.New("unidad no encontrada")
	// ErrUnidadInvalid: payload inválido.
	ErrUnidadInvalid = errors.New("unidad inválida")
	// ErrDuplicateCodigo: ya existe una UF con el mismo código en el consorcio.
	ErrDuplicateCodigo = errors.New("ya existe una unidad con ese código")
	// ErrDuplicateDocumento: otra persona ocupa ese documento en el tenant.
	ErrDuplicateDocumento = errors.New("ya existe una persona con ese documento")
	// ErrVinculoInvalid: vínculo, porcentaje o vigencia inválidos.
	ErrVinculoInvalid = errors.New("vínculo inválido")
)

var (
	coefRe        = regexp.MustCompile(`^([0-9]+(\.[0-9]{1,8})?|0)$`)
	pctRe         = regexp.MustCompile(`^([0-9]+(\.[0-9]{1,2})?|0)$`)
	dateRe        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	validTiposUF  = map[string]bool{"departamento": true, "cochera": true, "local": true, "unidad_edificio": true, "otros": true}
	validVinculos = map[string]bool{"propietario": true, "inquilino": true, "apoderado": true}
)

// PersonaDTO es la representación pública de una persona (contrato HTTP).
type PersonaDTO struct {
	ID        string  `json:"id"`
	Nombre    string  `json:"nombre"`
	Documento *string `json:"documento"`
	Email     *string `json:"email"`
	Telefono  *string `json:"telefono"`
}

// PersonaVinculoDTO es un vínculo vigente con la persona resuelta.
type PersonaVinculoDTO struct {
	Persona    PersonaDTO `json:"persona"`
	Vinculo    string     `json:"vinculo"`
	Porcentaje *string    `json:"porcentaje"`
	ValidFrom  string     `json:"valid_from"`
}

// UnidadDTO es la representación pública de una UF (contrato HTTP).
type UnidadDTO struct {
	ID          string              `json:"id"`
	ConsorcioID string              `json:"consorcio_id"`
	Codigo      string              `json:"codigo"`
	Tipo        string              `json:"tipo"`
	Superficie  *float64            `json:"superficie"`
	Coeficiente string              `json:"coeficiente"`
	Estado      string              `json:"estado"`
	Personas    []PersonaVinculoDTO `json:"personas"`
}

// PersonaInput describe la persona a vincular al crear la UF.
type PersonaInput struct {
	Nombre    string  `json:"nombre"`
	Documento *string `json:"documento"`
	Email     *string `json:"email"`
	Telefono  *string `json:"telefono"`
}

// PersonaVinculoInput es un vínculo propuesto al crear la UF.
type PersonaVinculoInput struct {
	Persona    PersonaInput `json:"persona"`
	Vinculo    string       `json:"vinculo"`
	Porcentaje *string      `json:"porcentaje"`
	ValidFrom  *string      `json:"valid_from"`
}

// UnidadInput es el payload de creación de una UF (contrato HTTP).
type UnidadInput struct {
	Codigo      string                `json:"codigo"`
	Tipo        string                `json:"tipo"`
	Superficie  *float64              `json:"superficie"`
	Coeficiente string                `json:"coeficiente"`
	Personas    []PersonaVinculoInput `json:"personas"`
}

// ListUnidades devuelve las UFs de un consorcio del tenant activo con sus
// vínculos vigentes.
func ListUnidades(ctx context.Context, q *db.Queries, consorcioID, estado string) ([]UnidadDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return nil, ErrUnidadInvalid
	}
	rows, err := q.ListUnidades(ctx, db.ListUnidadesParams{ConsorcioID: cid, Estado: estado})
	if err != nil {
		return nil, err
	}
	out := make([]UnidadDTO, 0, len(rows))
	for _, r := range rows {
		dto := unidadToDTO(r)
		vinculos, err := q.ListVinculosVigentes(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		dto.Personas = make([]PersonaVinculoDTO, 0, len(vinculos))
		for _, v := range vinculos {
			dto.Personas = append(dto.Personas, vinculoToDTO(v))
		}
		out = append(out, dto)
	}
	return out, nil
}

// GetUnidad devuelve una UF del consorcio con sus vínculos vigentes.
func GetUnidad(ctx context.Context, q *db.Queries, consorcioID, unidadID string) (UnidadDTO, error) {
	var cid, uid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return UnidadDTO{}, ErrUnidadInvalid
	}
	if err := uid.Scan(unidadID); err != nil {
		return UnidadDTO{}, ErrUnidadInvalid
	}
	u, err := q.GetUnidad(ctx, db.GetUnidadParams{ConsorcioID: cid, ID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return UnidadDTO{}, ErrUnidadNotFound
	}
	if err != nil {
		return UnidadDTO{}, err
	}
	dto := unidadToDTO(u)
	vinculos, err := q.ListVinculosVigentes(ctx, u.ID)
	if err != nil {
		return UnidadDTO{}, err
	}
	dto.Personas = make([]PersonaVinculoDTO, 0, len(vinculos))
	for _, v := range vinculos {
		dto.Personas = append(dto.Personas, vinculoToDTO(v))
	}
	return dto, nil
}

// CreateUnidad valida y crea una UF con sus vínculos iniciales. Las personas se
// reutilizan por documento dentro del tenant; si no existen, se crean.
func CreateUnidad(ctx context.Context, q *db.Queries, consorcioID string, in UnidadInput) (UnidadDTO, error) {
	u, err := validateUnidad(in, true)
	if err != nil {
		return UnidadDTO{}, err
	}
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return UnidadDTO{}, ErrUnidadInvalid
	}

	created, err := q.CreateUnidad(ctx, db.CreateUnidadParams{
		ConsorcioID: cid,
		Codigo:      u.codigo,
		Tipo:        u.tipo,
		Superficie:  u.superficie,
		Coeficiente: u.coeficiente,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return UnidadDTO{}, ErrDuplicateCodigo
		}
		return UnidadDTO{}, err
	}

	for _, v := range u.vinculos {
		if err := attachPersona(ctx, q, created.ID, v); err != nil {
			return UnidadDTO{}, err
		}
	}

	return GetUnidad(ctx, q, consorcioID, created.ID.String())
}

// attachPersona reusa/crea la persona y registra el vínculo vigente.
func attachPersona(ctx context.Context, q *db.Queries, unidadID pgtype.UUID, v validatedVinculo) error {
	var persona db.Persona
	if v.documento != "" {
		p, err := q.GetPersonaByDocumento(ctx, v.documento)
		if err == nil {
			persona = p
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	if !persona.ID.Valid || persona.ID.Bytes == [16]byte{} {
		p, err := q.CreatePersona(ctx, db.CreatePersonaParams{
			Nombre:    v.nombre,
			Documento: textOrNil(v.documento),
			Email:     textOrNil(v.email),
			Telefono:  textOrNil(v.telefono),
		})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicateDocumento
			}
			return err
		}
		persona = p
	}

	// Vínculo histórico: si la persona ya está vinculada a la UF con el mismo rol,
	// se cierra el vigente y se crea otro (nunca se sobreescribe).
	vigente, err := q.VinculoVigentePorPersona(ctx, db.VinculoVigentePorPersonaParams{
		UnidadID:  unidadID,
		PersonaID: persona.ID,
		Vinculo:   v.vinculo,
	})
	switch {
	case err == nil:
		if _, err := q.CloseVinculoVigente(ctx, db.CloseVinculoVigenteParams{
			UnidadID:  unidadID,
			PersonaID: persona.ID,
			ValidTo:   pgtype.Date{Time: time.Now(), Valid: true},
		}); err != nil {
			return err
		}
		_ = vigente
	case !errors.Is(err, pgx.ErrNoRows):
		return err
	}

	_, err = q.CreateVinculo(ctx, db.CreateVinculoParams{
		UnidadID:   unidadID,
		PersonaID:  persona.ID,
		Vinculo:    v.vinculo,
		Porcentaje: v.porcentaje,
		ValidFrom:  v.validFrom,
	})
	return err
}

type validatedUnidad struct {
	codigo      string
	tipo        string
	superficie  pgtype.Numeric
	coeficiente pgtype.Numeric
	vinculos    []validatedVinculo
}

type validatedVinculo struct {
	nombre     string
	documento  string
	email      string
	telefono   string
	vinculo    string
	porcentaje pgtype.Numeric
	validFrom  pgtype.Date
}

func validateUnidad(in UnidadInput, requireBasicos bool) (validatedUnidad, error) {
	var out validatedUnidad
	if requireBasicos {
		if strings.TrimSpace(in.Codigo) == "" {
			return out, ErrUnidadInvalid
		}
	}
	code := strings.TrimSpace(in.Codigo)
	if code == "" || len(code) > 50 {
		return out, ErrUnidadInvalid
	}
	if !validTiposUF[in.Tipo] {
		return out, ErrUnidadInvalid
	}
	out.codigo = code
	out.tipo = in.Tipo

	if in.Superficie != nil && *in.Superficie < 0 {
		return out, ErrUnidadInvalid
	}
	if in.Superficie != nil {
		out.superficie = floatToNumeric(*in.Superficie)
	}

	if in.Coeficiente == "" {
		out.coeficiente = pgtype.Numeric{Int: big.NewInt(0), Valid: true}
	} else {
		n, err := parseNumeric(in.Coeficiente)
		if err != nil {
			return out, ErrUnidadInvalid
		}
		out.coeficiente = n
	}

	for _, p := range in.Personas {
		v, err := validateVinculo(p)
		if err != nil {
			return out, err
		}
		out.vinculos = append(out.vinculos, v)
	}
	return out, nil
}

func validateVinculo(in PersonaVinculoInput) (validatedVinculo, error) {
	var out validatedVinculo
	nombre := strings.TrimSpace(in.Persona.Nombre)
	if nombre == "" || len(nombre) > 200 {
		return out, ErrVinculoInvalid
	}
	out.nombre = nombre
	if in.Persona.Documento != nil {
		out.documento = strings.TrimSpace(*in.Persona.Documento)
	}
	if in.Persona.Email != nil {
		out.email = strings.TrimSpace(*in.Persona.Email)
	}
	if in.Persona.Telefono != nil {
		out.telefono = strings.TrimSpace(*in.Persona.Telefono)
	}
	if !validVinculos[in.Vinculo] {
		return out, ErrVinculoInvalid
	}
	out.vinculo = in.Vinculo

	if in.Porcentaje != nil {
		if !pctRe.MatchString(*in.Porcentaje) {
			return out, ErrVinculoInvalid
		}
		n, err := parseNumeric(*in.Porcentaje)
		if err != nil {
			return out, ErrVinculoInvalid
		}
		if f, err := n.Float64Value(); err != nil || f.Valid && f.Float64 > 100 {
			return out, ErrVinculoInvalid
		}
		out.porcentaje = n
	}

	if in.ValidFrom != nil {
		if !dateRe.MatchString(*in.ValidFrom) {
			return out, ErrVinculoInvalid
		}
		t, err := time.Parse("2006-01-02", *in.ValidFrom)
		if err != nil {
			return out, ErrVinculoInvalid
		}
		out.validFrom = pgtype.Date{Time: t, Valid: true}
	} else {
		out.validFrom = pgtype.Date{Time: time.Now(), Valid: true}
	}
	return out, nil
}

func unidadToDTO(u db.Unidade) UnidadDTO {
	dto := UnidadDTO{
		ID:          u.ID.String(),
		ConsorcioID: u.ConsorcioID.String(),
		Codigo:      u.Codigo,
		Tipo:        u.Tipo,
		Coeficiente: numericToString(u.Coeficiente),
		Estado:      u.Estado,
		Personas:    []PersonaVinculoDTO{},
	}
	if f, err := u.Superficie.Float64Value(); err == nil && f.Valid {
		v := f.Float64
		dto.Superficie = &v
	}
	return dto
}

func vinculoToDTO(v db.ListVinculosVigentesRow) PersonaVinculoDTO {
	dto := PersonaVinculoDTO{
		Persona: PersonaDTO{
			ID:     v.PersonaID.String(),
			Nombre: v.Nombre,
		},
		Vinculo:   v.Vinculo,
		ValidFrom: v.ValidFrom.Time.Format("2006-01-02"),
	}
	if v.Documento.Valid {
		s := v.Documento.String
		dto.Persona.Documento = &s
	}
	if v.Email.Valid {
		s := v.Email.String
		dto.Persona.Email = &s
	}
	if v.Telefono.Valid {
		s := v.Telefono.String
		dto.Persona.Telefono = &s
	}
	if v.Porcentaje.Valid {
		s := numericToString(v.Porcentaje)
		dto.Porcentaje = &s
	}
	return dto
}

// parseNumeric convierte un string decimal a pgtype.Numeric con precisión exacta.
func parseNumeric(s string) (pgtype.Numeric, error) {
	if !coefRe.MatchString(s) {
		return pgtype.Numeric{}, ErrUnidadInvalid
	}
	neg := strings.HasPrefix(s, "-")
	body := strings.TrimPrefix(s, "-")
	parts := strings.SplitN(body, ".", 2)
	digits := parts[0]
	exp := int32(0)
	if len(parts) == 2 {
		digits += parts[1]
		exp = -int32(len(parts[1]))
	}
	bigInt, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return pgtype.Numeric{}, ErrUnidadInvalid
	}
	if neg {
		bigInt.Neg(bigInt)
	}
	return pgtype.Numeric{Int: bigInt, Exp: exp, Valid: true}, nil
}

// numericToString serializa un pgtype.Numeric como texto decimal (no JSON).
func numericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return ""
	}
	if f, err := n.Float64Value(); err == nil && f.Valid {
		return strconv.FormatFloat(f.Float64, 'f', -1, 64)
	}
	return "0"
}

// floatToNumeric aproxima un float64 a pgtype.Numeric (superficie, NUMERIC(12,4)).
func floatToNumeric(f float64) pgtype.Numeric {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if n, err := parseNumeric(s); err == nil {
		return n
	}
	return pgtype.Numeric{Int: new(big.Int).SetInt64(int64(f)), Valid: true}
}
