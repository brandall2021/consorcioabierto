package reclamos

import (
	"context"
	"errors"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrReclamoInvalid  = errors.New("datos de reclamo inválidos")
	ErrReclamoNotFound = errors.New("reclamo no encontrado")
)

type ReclamoInput struct {
	UnidadID  string
	Categoria string
	Texto     string
}

type MessageInput struct {
	Texto     string
	AdjuntoID *string
}

type TransitionInput struct {
	Accion        string
	Motivo        *string
	ResponsableID *string
}

type ReclamoDTO struct {
	ID            string    `json:"id"`
	ConsorcioID   string    `json:"consorcio_id"`
	UnidadID      string    `json:"unidad_id"`
	Categoria     string    `json:"categoria"`
	Texto         string    `json:"texto"`
	Estado        string    `json:"estado"`
	ResponsableID *string   `json:"responsable_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ReclamoMensajeDTO struct {
	ID        string    `json:"id"`
	ReclamoID string    `json:"reclamo_id"`
	Texto     string    `json:"texto"`
	AdjuntoID *string   `json:"adjunto_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ReclamoTransicionDTO struct {
	ID            string    `json:"id"`
	ReclamoID     string    `json:"reclamo_id"`
	Accion        string    `json:"accion"`
	Motivo        *string   `json:"motivo"`
	ResponsableID *string   `json:"responsable_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type ReclamoDetalleDTO struct {
	Reclamo      ReclamoDTO             `json:"reclamo"`
	Mensajes     []ReclamoMensajeDTO    `json:"mensajes"`
	Transiciones []ReclamoTransicionDTO `json:"transiciones"`
}

func ListReclamos(ctx context.Context, q *db.Queries, consorcioID string) ([]ReclamoDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(consorcioID)); err != nil {
		return nil, ErrReclamoInvalid
	}
	rows, err := q.ListReclamos(ctx, cid)
	if err != nil {
		return nil, err
	}
	out := make([]ReclamoDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, reclamoDTO(row))
	}
	return out, nil
}

func GetReclamoDetalle(ctx context.Context, q *db.Queries, reclamoID string) (ReclamoDetalleDTO, error) {
	var rid pgtype.UUID
	if err := rid.Scan(strings.TrimSpace(reclamoID)); err != nil {
		return ReclamoDetalleDTO{}, ErrReclamoInvalid
	}
	row, err := q.GetReclamo(ctx, rid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReclamoDetalleDTO{}, ErrReclamoNotFound
		}
		return ReclamoDetalleDTO{}, err
	}
	mensajes, err := q.ListReclamoMensajes(ctx, rid)
	if err != nil {
		return ReclamoDetalleDTO{}, err
	}
	transiciones, err := q.ListReclamoTransiciones(ctx, rid)
	if err != nil {
		return ReclamoDetalleDTO{}, err
	}
	return ReclamoDetalleDTO{
		Reclamo:      reclamoDTO(row),
		Mensajes:     reclamoMensajesDTO(mensajes),
		Transiciones: reclamoTransicionesDTO(transiciones),
	}, nil
}

func CreateReclamo(ctx context.Context, q *db.Queries, consorcioID, createdBy string, in ReclamoInput) (ReclamoDTO, error) {
	if strings.TrimSpace(in.UnidadID) == "" || strings.TrimSpace(in.Categoria) == "" || strings.TrimSpace(in.Texto) == "" {
		return ReclamoDTO{}, ErrReclamoInvalid
	}
	var cid, uid, actor pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(consorcioID)); err != nil {
		return ReclamoDTO{}, ErrReclamoInvalid
	}
	if err := uid.Scan(strings.TrimSpace(in.UnidadID)); err != nil {
		return ReclamoDTO{}, ErrReclamoInvalid
	}
	if err := actor.Scan(strings.TrimSpace(createdBy)); err != nil {
		return ReclamoDTO{}, ErrReclamoInvalid
	}
	row, err := q.CreateReclamo(ctx, db.CreateReclamoParams{
		ConsorcioID: cid,
		UnidadID:    uid,
		Categoria:   strings.TrimSpace(in.Categoria),
		Texto:       strings.TrimSpace(in.Texto),
		CreatedBy:   actor,
	})
	if err != nil {
		return ReclamoDTO{}, err
	}
	return reclamoDTO(row), nil
}

func AddReclamoMensaje(ctx context.Context, q *db.Queries, reclamoID, createdBy string, in MessageInput) (ReclamoMensajeDTO, error) {
	if strings.TrimSpace(in.Texto) == "" {
		return ReclamoMensajeDTO{}, ErrReclamoInvalid
	}
	var rid, actor, adjunto pgtype.UUID
	if err := rid.Scan(strings.TrimSpace(reclamoID)); err != nil {
		return ReclamoMensajeDTO{}, ErrReclamoInvalid
	}
	if err := actor.Scan(strings.TrimSpace(createdBy)); err != nil {
		return ReclamoMensajeDTO{}, ErrReclamoInvalid
	}
	if in.AdjuntoID != nil && strings.TrimSpace(*in.AdjuntoID) != "" {
		if err := adjunto.Scan(strings.TrimSpace(*in.AdjuntoID)); err != nil {
			return ReclamoMensajeDTO{}, ErrReclamoInvalid
		}
	}
	row, err := q.CreateReclamoMensaje(ctx, db.CreateReclamoMensajeParams{
		ReclamoID: rid,
		Texto:     strings.TrimSpace(in.Texto),
		AdjuntoID: adjunto,
		CreatedBy: actor,
	})
	if err != nil {
		return ReclamoMensajeDTO{}, err
	}
	return reclamoMensajeDTO(row), nil
}

func TransicionarReclamo(ctx context.Context, q *db.Queries, reclamoID, createdBy string, in TransitionInput) (ReclamoDTO, error) {
	if strings.TrimSpace(in.Accion) == "" {
		return ReclamoDTO{}, ErrReclamoInvalid
	}
	var rid, actor, responsable pgtype.UUID
	if err := rid.Scan(strings.TrimSpace(reclamoID)); err != nil {
		return ReclamoDTO{}, ErrReclamoInvalid
	}
	if in.ResponsableID != nil && strings.TrimSpace(*in.ResponsableID) != "" {
		if err := responsable.Scan(strings.TrimSpace(*in.ResponsableID)); err != nil {
			return ReclamoDTO{}, ErrReclamoInvalid
		}
	}
	if err := actor.Scan(strings.TrimSpace(createdBy)); err != nil {
		return ReclamoDTO{}, ErrReclamoInvalid
	}
	current, err := q.GetReclamo(ctx, rid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReclamoDTO{}, ErrReclamoNotFound
		}
		return ReclamoDTO{}, err
	}
	nextEstado := nextState(current.Estado, strings.TrimSpace(in.Accion))
	updated, err := q.UpdateReclamo(ctx, db.UpdateReclamoParams{
		ID:            rid,
		Estado:        nullableState(nextEstado),
		ResponsableID: nullableUUID(in.ResponsableID, responsable),
	})
	if err != nil {
		return ReclamoDTO{}, err
	}
	if _, err := q.CreateReclamoTransicion(ctx, db.CreateReclamoTransicionParams{
		ReclamoID:     rid,
		Accion:        strings.TrimSpace(in.Accion),
		Motivo:        textValue(in.Motivo),
		ResponsableID: nullableUUID(in.ResponsableID, responsable),
		CreatedBy:     actor,
	}); err != nil {
		return ReclamoDTO{}, err
	}
	return reclamoDTO(updated), nil
}

func reclamoDTO(row db.Reclamo) ReclamoDTO {
	return ReclamoDTO{
		ID:            row.ID.String(),
		ConsorcioID:   row.ConsorcioID.String(),
		UnidadID:      row.UnidadID.String(),
		Categoria:     row.Categoria,
		Texto:         row.Texto,
		Estado:        row.Estado,
		ResponsableID: uuidPtr(row.ResponsableID),
		CreatedAt:     row.CreatedAt.Time.UTC(),
		UpdatedAt:     row.UpdatedAt.Time.UTC(),
	}
}

func reclamoMensajesDTO(rows []db.ReclamoMensaje) []ReclamoMensajeDTO {
	out := make([]ReclamoMensajeDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, reclamoMensajeDTO(row))
	}
	return out
}

func reclamoMensajeDTO(row db.ReclamoMensaje) ReclamoMensajeDTO {
	return ReclamoMensajeDTO{
		ID:        row.ID.String(),
		ReclamoID: row.ReclamoID.String(),
		Texto:     row.Texto,
		AdjuntoID: uuidPtr(row.AdjuntoID),
		CreatedAt: row.CreatedAt.Time.UTC(),
	}
}

func reclamoTransicionesDTO(rows []db.ReclamoTransicione) []ReclamoTransicionDTO {
	out := make([]ReclamoTransicionDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, ReclamoTransicionDTO{
			ID:            row.ID.String(),
			ReclamoID:     row.ReclamoID.String(),
			Accion:        row.Accion,
			Motivo:        textPtr(row.Motivo),
			ResponsableID: uuidPtr(row.ResponsableID),
			CreatedAt:     row.CreatedAt.Time.UTC(),
		})
	}
	return out
}

func nextState(current, accion string) string {
	switch accion {
	case "asignar":
		return current
	case "en_progreso":
		return "en_progreso"
	case "resolver":
		return "resuelto"
	case "cerrar":
		return "cerrado"
	case "reabrir":
		return "abierto"
	default:
		return current
	}
}

func nullableState(v string) pgtype.Text {
	if strings.TrimSpace(v) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func nullableUUID(src *string, fallback pgtype.UUID) pgtype.UUID {
	if src != nil && strings.TrimSpace(*src) != "" {
		var u pgtype.UUID
		_ = u.Scan(strings.TrimSpace(*src))
		return u
	}
	return fallback
}

func textValue(src *string) pgtype.Text {
	if src == nil || strings.TrimSpace(*src) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strings.TrimSpace(*src), Valid: true}
}

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func uuidPtr(v pgtype.UUID) *string {
	if v == (pgtype.UUID{}) {
		return nil
	}
	s := v.String()
	return &s
}
