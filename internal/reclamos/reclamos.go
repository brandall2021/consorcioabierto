// Package reclamos implementa el dominio de reclamos [H5.3]: creación,
// conversación, máquina de estados (abierto → en_progreso → resuelto → cerrado)
// con reapertura y historial, y SLA.
package reclamos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	EstadoAbierto    = "abierto"
	EstadoEnProgreso = "en_progreso"
	EstadoResuelto   = "resuelto"
	EstadoCerrado    = "cerrado"
)

const (
	AccionAsignar    = "asignar"
	AccionEnProgreso = "en_progreso"
	AccionResolver   = "resolver"
	AccionCerrar     = "cerrar"
	AccionReabrir    = "reabrir"
)

const slaDuracion = 72 * time.Hour

var ErrInvalidTransition = errors.New("transición inválida para el estado del reclamo")

var ErrInvalidID = errors.New("id inválido")

var transiciones = map[[2]string]string{
	{EstadoAbierto, AccionEnProgreso}:  EstadoEnProgreso,
	{EstadoAbierto, AccionResolver}:    EstadoResuelto,
	{EstadoAbierto, AccionCerrar}:      EstadoCerrado,
	{EstadoEnProgreso, AccionResolver}: EstadoResuelto,
	{EstadoEnProgreso, AccionCerrar}:   EstadoCerrado,
	{EstadoResuelto, AccionCerrar}:     EstadoCerrado,
	{EstadoResuelto, AccionReabrir}:    EstadoAbierto,
	{EstadoCerrado, AccionReabrir}:     EstadoAbierto,
}

var accionesReasignacion = map[string]bool{
	EstadoAbierto:    true,
	EstadoEnProgreso: true,
}

// Reclamo es el DTO del contrato OpenAPI (schema Reclamo).
type Reclamo struct {
	ID            string    `json:"id"`
	ConsorcioID   string    `json:"consorcio_id"`
	UnidadID      string    `json:"unidad_id"`
	Categoria     string    `json:"categoria"`
	Estado        string    `json:"estado"`
	ResponsableID *string   `json:"responsable_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// Mensaje es el DTO del contrato OpenAPI (schema ReclamoMensaje).
type Mensaje struct {
	ID        string    `json:"id"`
	ReclamoID string    `json:"reclamo_id"`
	Texto     string    `json:"texto"`
	AdjuntoID *string   `json:"adjunto_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Transicion es el DTO del contrato OpenAPI (schema ReclamoTransicion).
type Transicion struct {
	ID            string    `json:"id"`
	ReclamoID     string    `json:"reclamo_id"`
	Accion        string    `json:"accion"`
	Motivo        *string   `json:"motivo"`
	ResponsableID *string   `json:"responsable_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// Detalle es el DTO del contrato OpenAPI (schema ReclamoDetalle).
type Detalle struct {
	Reclamo      Reclamo      `json:"reclamo"`
	Mensajes     []Mensaje    `json:"mensajes"`
	Transiciones []Transicion `json:"transiciones"`
}

type CreateInput struct {
	UnidadID  string `json:"unidad_id"`
	Categoria string `json:"categoria"`
	Texto     string `json:"texto"`
}

type MensajeInput struct {
	Texto     string  `json:"texto"`
	AdjuntoID *string `json:"adjunto_id"`
}

type TransicionInput struct {
	Accion        string  `json:"accion"`
	Motivo        *string `json:"motivo"`
	ResponsableID *string `json:"responsable_id"`
}

type Queryer interface {
	ListReclamos(context.Context, db.ListReclamosParams) ([]db.Reclamo, error)
	GetReclamo(context.Context, pgtype.UUID) (db.Reclamo, error)
	CreateReclamo(context.Context, db.CreateReclamoParams) (db.Reclamo, error)
	UpdateReclamoEstado(context.Context, db.UpdateReclamoEstadoParams) (db.Reclamo, error)
	ListReclamoMensajes(context.Context, pgtype.UUID) ([]db.ReclamoMensaje, error)
	InsertReclamoMensaje(context.Context, db.InsertReclamoMensajeParams) (db.ReclamoMensaje, error)
	ListReclamoTransiciones(context.Context, pgtype.UUID) ([]db.ReclamoTransicione, error)
	InsertReclamoTransicion(context.Context, db.InsertReclamoTransicionParams) (db.ReclamoTransicione, error)
	GetDocumento(context.Context, pgtype.UUID) (db.Documento, error)
}

func List(ctx context.Context, q Queryer, consorcioID, estado string) ([]Reclamo, error) {
	consorcio, err := parseUUID(consorcioID)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListReclamos(ctx, db.ListReclamosParams{ConsorcioID: consorcio, Estado: strings.TrimSpace(estado)})
	if err != nil {
		return nil, err
	}
	items := make([]Reclamo, 0, len(rows))
	for _, row := range rows {
		items = append(items, toReclamo(row))
	}
	return items, nil
}

func Create(ctx context.Context, q Queryer, consorcioID string, in CreateInput) (Reclamo, error) {
	if strings.TrimSpace(in.Categoria) == "" {
		return Reclamo{}, errors.New("categoria es obligatoria")
	}
	if len([]rune(strings.TrimSpace(in.Texto))) < 3 {
		return Reclamo{}, errors.New("texto debe tener al menos 3 caracteres")
	}
	consorcio, err := parseUUID(consorcioID)
	if err != nil {
		return Reclamo{}, err
	}
	unidad, err := parseUUID(strings.TrimSpace(in.UnidadID))
	if err != nil {
		return Reclamo{}, err
	}
	row, err := q.CreateReclamo(ctx, db.CreateReclamoParams{
		ConsorcioID: consorcio,
		UnidadID:    unidad,
		Categoria:   strings.TrimSpace(in.Categoria),
		Texto:       strings.TrimSpace(in.Texto),
		SlaDueAt:    pgtype.Timestamptz{Time: time.Now().UTC().Add(slaDuracion), Valid: true},
	})
	if err != nil {
		return Reclamo{}, err
	}
	return toReclamo(row), nil
}

func Get(ctx context.Context, q Queryer, reclamoID string) (Detalle, error) {
	id, err := parseUUID(reclamoID)
	if err != nil {
		return Detalle{}, err
	}
	row, err := q.GetReclamo(ctx, id)
	if err != nil {
		return Detalle{}, err
	}
	mensajes, err := q.ListReclamoMensajes(ctx, id)
	if err != nil {
		return Detalle{}, err
	}
	transiciones, err := q.ListReclamoTransiciones(ctx, id)
	if err != nil {
		return Detalle{}, err
	}
	detalle := Detalle{
		Reclamo:      toReclamo(row),
		Mensajes:     make([]Mensaje, 0, len(mensajes)),
		Transiciones: make([]Transicion, 0, len(transiciones)),
	}
	for _, m := range mensajes {
		detalle.Mensajes = append(detalle.Mensajes, toMensaje(m))
	}
	for _, tr := range transiciones {
		detalle.Transiciones = append(detalle.Transiciones, toTransicion(tr))
	}
	return detalle, nil
}

func AddMensaje(ctx context.Context, q Queryer, reclamoID string, in MensajeInput) (Mensaje, error) {
	texto := strings.TrimSpace(in.Texto)
	if texto == "" {
		return Mensaje{}, errors.New("texto es obligatorio")
	}
	id, err := parseUUID(reclamoID)
	if err != nil {
		return Mensaje{}, err
	}
	adjunto, err := optionalUUID(in.AdjuntoID)
	if err != nil {
		return Mensaje{}, err
	}
	if adjunto.Valid {
		if _, err := q.GetDocumento(ctx, adjunto); err != nil {
			return Mensaje{}, errors.New("adjunto inexistente")
		}
	}
	row, err := q.InsertReclamoMensaje(ctx, db.InsertReclamoMensajeParams{
		ReclamoID: id,
		Texto:     texto,
		AdjuntoID: adjunto,
	})
	if err != nil {
		return Mensaje{}, err
	}
	return toMensaje(row), nil
}

func Transicionar(ctx context.Context, q Queryer, reclamoID string, in TransicionInput) (Reclamo, error) {
	id, err := parseUUID(reclamoID)
	if err != nil {
		return Reclamo{}, err
	}
	accion := strings.ToLower(strings.TrimSpace(in.Accion))
	motivo := strings.TrimSpace(optionalText(in.Motivo))
	responsable, err := optionalUUID(in.ResponsableID)
	if err != nil {
		return Reclamo{}, err
	}

	row, err := q.GetReclamo(ctx, id)
	if err != nil {
		return Reclamo{}, err
	}

	toEstado, ok := transiciones[[2]string{row.Estado, accion}]
	if accion == AccionAsignar {
		if !accionesReasignacion[row.Estado] {
			return Reclamo{}, ErrInvalidTransition
		}
		if !responsable.Valid {
			return Reclamo{}, errors.New("asignar requiere responsable_id")
		}
		toEstado = row.Estado
		ok = true
	}
	if !ok {
		return Reclamo{}, ErrInvalidTransition
	}
	if accion == AccionReabrir && motivo == "" {
		return Reclamo{}, errors.New("reabrir requiere motivo")
	}
	if !responsable.Valid {
		responsable = row.ResponsableID
	}

	if _, err := q.InsertReclamoTransicion(ctx, db.InsertReclamoTransicionParams{
		ReclamoID:     id,
		Accion:        accion,
		Motivo:        pgtype.Text{String: motivo, Valid: motivo != ""},
		FromEstado:    row.Estado,
		ToEstado:      toEstado,
		ResponsableID: responsable,
	}); err != nil {
		return Reclamo{}, err
	}
	updated, err := q.UpdateReclamoEstado(ctx, db.UpdateReclamoEstadoParams{
		ID:            id,
		Estado:        toEstado,
		ResponsableID: responsable,
	})
	if err != nil {
		return Reclamo{}, err
	}
	return toReclamo(updated), nil
}

func toReclamo(row db.Reclamo) Reclamo {
	out := Reclamo{
		ID:          row.ID.String(),
		ConsorcioID: row.ConsorcioID.String(),
		UnidadID:    row.UnidadID.String(),
		Categoria:   row.Categoria,
		Estado:      row.Estado,
		CreatedAt:   row.CreatedAt.Time,
	}
	if row.ResponsableID.Valid {
		s := row.ResponsableID.String()
		out.ResponsableID = &s
	}
	return out
}

func toMensaje(row db.ReclamoMensaje) Mensaje {
	out := Mensaje{
		ID:        row.ID.String(),
		ReclamoID: row.ReclamoID.String(),
		Texto:     row.Texto,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.AdjuntoID.Valid {
		s := row.AdjuntoID.String()
		out.AdjuntoID = &s
	}
	return out
}

func toTransicion(row db.ReclamoTransicione) Transicion {
	out := Transicion{
		ID:        row.ID.String(),
		ReclamoID: row.ReclamoID.String(),
		Accion:    row.Accion,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.Motivo.Valid {
		s := row.Motivo.String
		out.Motivo = &s
	}
	if row.ResponsableID.Valid {
		s := row.ResponsableID.String()
		out.ResponsableID = &s
	}
	return out
}

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: %v", ErrInvalidID, err)
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func optionalUUID(s *string) (pgtype.UUID, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return pgtype.UUID{}, nil
	}
	return parseUUID(strings.TrimSpace(*s))
}

func optionalText(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
