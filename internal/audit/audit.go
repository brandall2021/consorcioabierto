// Package audit registra eventos append-only de acciones de negocio
// (audit_events) con contexto de auditoría: actor, tenant, acción, recurso,
// request ID, IP y user-agent ([ADR-0009], §5.4 de la especificación).
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event es un evento de auditoría pendiente de persistir.
type Event struct {
	TenantID        string
	ActorID         string
	ActorMembership string
	Accion          string
	RecursoType     string
	RecursoID       string
	RequestID       string
	IP              string
	UserAgent       string
	Diff            map[string]any
}

// Recorder persiste eventos de auditoría de forma append-only.
type Recorder struct {
	q *db.Queries
}

// New crea un Recorder sobre el pool compartido.
func New(pool *pgxpool.Pool) *Recorder {
	return &Recorder{q: db.New(pool)}
}

// Record inserta un evento vía la función SECURITY DEFINER. Es best-effort:
// un error de auditoría no debe romper la operación de negocio que la originó
// (se registra y se continúa).
func (r *Recorder) Record(ctx context.Context, e Event) error {
	if e.Accion == "" || e.RecursoType == "" {
		return nil
	}
	var diff []byte
	if e.Diff != nil {
		b, err := json.Marshal(e.Diff)
		if err != nil {
			return err
		}
		diff = b
	}
	_, err := r.q.RecordAuditEvent(ctx, db.RecordAuditEventParams{
		TenantID:        uuidOrNil(e.TenantID),
		ActorID:         uuidOrNil(e.ActorID),
		ActorMembership: uuidOrNil(e.ActorMembership),
		Accion:          e.Accion,
		RecursoType:     e.RecursoType,
		RecursoID:       e.RecursoID,
		RequestID:       e.RequestID,
		Ip:              e.IP,
		UserAgent:       e.UserAgent,
		Diff:            diff,
	})
	return err
}

// Filter acota la lista de eventos de auditoría.
type Filter struct {
	Accion      string
	RecursoType string
	Desde       *time.Time
	Hasta       *time.Time
	Cursor      *Cursor
	Limit       int
}

// Cursor es la clave de paginación keyset (created_at + id del último evento).
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// EventDTO es la representación pública de un evento (contracto HTTP): los
// campos pgtype y el diff JSONB se convierten a tipos serializables.
type EventDTO struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	ActorID         string          `json:"actor_id"`
	ActorMembership string          `json:"actor_membership"`
	Accion          string          `json:"accion"`
	RecursoType     string          `json:"recurso_type"`
	RecursoID       string          `json:"recurso_id"`
	RequestID       string          `json:"request_id"`
	IP              string          `json:"ip"`
	UserAgent       string          `json:"user_agent"`
	Diff            json.RawMessage `json:"diff"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ToDTO convierte un evento persistido a su representación pública.
func ToDTO(e db.AuditEvent) EventDTO {
	dto := EventDTO{
		ID:          e.ID.String(),
		Accion:      e.Accion,
		RecursoType: e.RecursoType,
		Diff:        e.Diff,
		CreatedAt:   e.CreatedAt.Time,
	}
	if e.TenantID.Valid {
		dto.TenantID = e.TenantID.String()
	}
	if e.ActorID.Valid {
		dto.ActorID = e.ActorID.String()
	}
	if e.ActorMembership.Valid {
		dto.ActorMembership = e.ActorMembership.String()
	}
	if e.RecursoID.Valid {
		dto.RecursoID = e.RecursoID.String
	}
	if e.RequestID.Valid {
		dto.RequestID = e.RequestID.String
	}
	if e.Ip.Valid {
		dto.IP = e.Ip.String
	}
	if e.UserAgent.Valid {
		dto.UserAgent = e.UserAgent.String
	}
	return dto
}

// List devuelve eventos de auditoría del tenant activo, filtrados y paginados
// por keyset. q debe ejecutarse dentro de una transacción con el contexto de
// RLS del tenant ya seteado (app.current_tenant_id()).
func (r *Recorder) List(ctx context.Context, q *db.Queries, f Filter) ([]db.AuditEvent, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	params := db.ListAuditEventsParams{
		Accion:      f.Accion,
		RecursoType: f.RecursoType,
		PageLimit:   int32(f.Limit),
	}
	if f.Desde != nil {
		params.Desde = pgtype.Timestamptz{Time: *f.Desde, Valid: true}
	}
	if f.Hasta != nil {
		params.Hasta = pgtype.Timestamptz{Time: *f.Hasta, Valid: true}
	}
	if f.Cursor != nil {
		params.CursorCreated = pgtype.Timestamptz{Time: f.Cursor.CreatedAt, Valid: true}
		_ = params.CursorID.Scan(f.Cursor.ID)
	}
	return q.ListAuditEvents(ctx, params)
}

func uuidOrNil(s string) pgtype.UUID {
	var u pgtype.UUID
	if s == "" {
		return u
	}
	_ = u.Scan(s)
	return u
}
