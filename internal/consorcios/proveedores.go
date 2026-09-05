// Proveedores por consorcio (H2.4, §5.4). CUIT único dentro del consorcio.
// Las funciones reciben un *db.Queries ligado a una transacción con el contexto
// de RLS ya seteado, igual que consorcios y unidades.
package consorcios

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
	// ErrProveedorNotFound: el proveedor no existe en el tenant activo.
	ErrProveedorNotFound = errors.New("proveedor no encontrado")
	// ErrProveedorInvalid: payload inválido (cuit/razon_social/contactos/estado).
	ErrProveedorInvalid = errors.New("proveedor inválido")
	// ErrProveedorDuplicateCuit: ya existe un proveedor con el mismo CUIT en el consorcio.
	ErrProveedorDuplicateCuit = errors.New("ya existe un proveedor con ese CUIT en el consorcio")
)

var validProveedorEstados = map[string]bool{"activo": true, "inactivo": true}

// ProveedorDTO es la representación pública de un proveedor (contrato HTTP).
type ProveedorDTO struct {
	ID               string    `json:"id"`
	ConsorcioID      string    `json:"consorcio_id"`
	Cuit             string    `json:"cuit"`
	RazonSocial      string    `json:"razon_social"`
	ContactoNombre   *string   `json:"contacto_nombre"`
	ContactoEmail    *string   `json:"contacto_email"`
	ContactoTelefono *string   `json:"contacto_telefono"`
	Estado           string    `json:"estado"`
	CreatedAt        time.Time `json:"created_at"`
}

// ProveedorFilter acota el listado de proveedores de un consorcio.
type ProveedorFilter struct {
	Q      string
	Estado string
}

// ProveedorInput es el payload de creación.
type ProveedorInput struct {
	Cuit             string  `json:"cuit"`
	RazonSocial      string  `json:"razon_social"`
	ContactoNombre   *string `json:"contacto_nombre"`
	ContactoEmail    *string `json:"contacto_email"`
	ContactoTelefono *string `json:"contacto_telefono"`
}

type validatedProveedor struct {
	cuit             string
	razonSocial      string
	contactoNombre   pgtype.Text
	contactoEmail    pgtype.Text
	contactoTelefono pgtype.Text
}

// ListProveedores devuelve los proveedores de un consorcio del tenant activo.
func ListProveedores(ctx context.Context, q *db.Queries, consorcioID string, f ProveedorFilter) ([]ProveedorDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return nil, ErrProveedorInvalid
	}
	rows, err := q.ListProveedores(ctx, db.ListProveedoresParams{
		ConsorcioID: cid,
		Q:           strings.TrimSpace(f.Q),
		Estado:      strings.TrimSpace(f.Estado),
	})
	if err != nil {
		return nil, err
	}
	out := make([]ProveedorDTO, 0, len(rows))
	for _, p := range rows {
		out = append(out, proveedorDTO(p))
	}
	return out, nil
}

// CreateProveedor valida y crea un proveedor en el consorcio del tenant activo.
func CreateProveedor(ctx context.Context, q *db.Queries, consorcioID string, in ProveedorInput) (ProveedorDTO, error) {
	validated, err := validateProveedor(in, true)
	if err != nil {
		return ProveedorDTO{}, err
	}
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return ProveedorDTO{}, ErrProveedorInvalid
	}

	created, err := q.CreateProveedor(ctx, db.CreateProveedorParams{
		ConsorcioID:      cid,
		Cuit:             validated.cuit,
		RazonSocial:      validated.razonSocial,
		ContactoNombre:   validated.contactoNombre,
		ContactoEmail:    validated.contactoEmail,
		ContactoTelefono: validated.contactoTelefono,
	})
	if err != nil {
		return ProveedorDTO{}, mapProveedorError(err)
	}
	return proveedorDTO(created), nil
}

// GetProveedor devuelve un proveedor del tenant activo por su id.
func GetProveedor(ctx context.Context, q *db.Queries, proveedorID string) (ProveedorDTO, error) {
	var pid pgtype.UUID
	if err := pid.Scan(proveedorID); err != nil {
		return ProveedorDTO{}, ErrProveedorNotFound
	}
	p, err := q.GetProveedor(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProveedorDTO{}, ErrProveedorNotFound
	}
	if err != nil {
		return ProveedorDTO{}, err
	}
	return proveedorDTO(p), nil
}

// UpdateProveedor actualiza los campos de un proveedor (requiere cuit y razon_social).
func UpdateProveedor(ctx context.Context, q *db.Queries, proveedorID string, in ProveedorInput) (ProveedorDTO, error) {
	validated, err := validateProveedor(in, true)
	if err != nil {
		return ProveedorDTO{}, err
	}
	var pid pgtype.UUID
	if err := pid.Scan(proveedorID); err != nil {
		return ProveedorDTO{}, ErrProveedorNotFound
	}
	updated, err := q.UpdateProveedor(ctx, db.UpdateProveedorParams{
		ID:               pid,
		Cuit:             validated.cuit,
		RazonSocial:      validated.razonSocial,
		ContactoNombre:   validated.contactoNombre,
		ContactoEmail:    validated.contactoEmail,
		ContactoTelefono: validated.contactoTelefono,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ProveedorDTO{}, ErrProveedorNotFound
	}
	if err != nil {
		return ProveedorDTO{}, mapProveedorError(err)
	}
	return proveedorDTO(updated), nil
}

// SetProveedorEstado activa/desactiva un proveedor (toggle de baja lógica).
func SetProveedorEstado(ctx context.Context, q *db.Queries, proveedorID, estado string) (ProveedorDTO, error) {
	estado = strings.TrimSpace(estado)
	if estado == "" {
		estado = "activo"
	}
	if !validProveedorEstados[estado] {
		return ProveedorDTO{}, ErrProveedorInvalid
	}
	var pid pgtype.UUID
	if err := pid.Scan(proveedorID); err != nil {
		return ProveedorDTO{}, ErrProveedorNotFound
	}
	updated, err := q.UpdateProveedorEstado(ctx, db.UpdateProveedorEstadoParams{ID: pid, Estado: estado})
	if errors.Is(err, pgx.ErrNoRows) {
		return ProveedorDTO{}, ErrProveedorNotFound
	}
	if err != nil {
		return ProveedorDTO{}, err
	}
	return proveedorDTO(updated), nil
}

func validateProveedor(in ProveedorInput, requireBasicos bool) (validatedProveedor, error) {
	v := validatedProveedor{
		cuit:        strings.TrimSpace(in.Cuit),
		razonSocial: strings.TrimSpace(in.RazonSocial),
	}
	// JSON null para campos opcionales se normaliza a nil; "" se trata como null.
	v.contactoNombre = textOrNil(ptrStr(normalizePtr(in.ContactoNombre)))
	v.contactoEmail = textOrNil(ptrStr(normalizePtr(in.ContactoEmail)))
	v.contactoTelefono = textOrNil(ptrStr(normalizePtr(in.ContactoTelefono)))

	if requireBasicos {
		if !cuitRe.MatchString(v.cuit) {
			return validatedProveedor{}, ErrProveedorInvalid
		}
		if v.razonSocial == "" || len(v.razonSocial) > 200 {
			return validatedProveedor{}, ErrProveedorInvalid
		}
	}
	if v.razonSocial != "" && len(v.razonSocial) > 200 {
		return validatedProveedor{}, ErrProveedorInvalid
	}
	if v.contactoEmail.Valid && len(v.contactoEmail.String) > 100 {
		return validatedProveedor{}, ErrProveedorInvalid
	}
	if v.contactoNombre.Valid && len(v.contactoNombre.String) > 200 {
		return validatedProveedor{}, ErrProveedorInvalid
	}
	if v.contactoTelefono.Valid && len(v.contactoTelefono.String) > 30 {
		return validatedProveedor{}, ErrProveedorInvalid
	}
	return v, nil
}

func normalizePtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// mapProveedorError traduce errores de unicidad del CUIT a un error de dominio.
func mapProveedorError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrProveedorDuplicateCuit
	}
	return err
}

func proveedorDTO(p db.Proveedore) ProveedorDTO {
	return ProveedorDTO{
		ID:               p.ID.String(),
		ConsorcioID:      p.ConsorcioID.String(),
		Cuit:             p.Cuit,
		RazonSocial:      p.RazonSocial,
		ContactoNombre:   textPtrOrNil(p.ContactoNombre),
		ContactoEmail:    textPtrOrNil(p.ContactoEmail),
		ContactoTelefono: textPtrOrNil(p.ContactoTelefono),
		Estado:           p.Estado,
		CreatedAt:        p.CreatedAt.Time,
	}
}

func textPtrOrNil(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}