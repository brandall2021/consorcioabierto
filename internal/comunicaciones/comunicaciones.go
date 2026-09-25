package comunicaciones

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrComunicadoInvalid   = errors.New("datos de comunicado inválidos")
	ErrComunicadoNotFound  = errors.New("comunicado no encontrado")
	ErrComunicadoPublished = errors.New("comunicado ya publicado")
)

const EventComunicadoPublished = "comunicado.publicado"

type ComunicadoInput struct {
	Titulo        string
	Cuerpo        string
	Destinatarios string
}

type ComunicadoDTO struct {
	ID            string     `json:"id"`
	ConsorcioID   string     `json:"consorcio_id"`
	Titulo        string     `json:"titulo"`
	Cuerpo        string     `json:"cuerpo"`
	Destinatarios string     `json:"destinatarios"`
	Estado        string     `json:"estado"`
	PublicadoAt   *time.Time `json:"publicado_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ComunicadoPayload struct {
	ComunicadoID  string `json:"comunicado_id"`
	ConsorcioID   string `json:"consorcio_id"`
	Titulo        string `json:"titulo"`
	Cuerpo        string `json:"cuerpo"`
	Destinatarios string `json:"destinatarios"`
}

func ListComunicados(ctx context.Context, q *db.Queries, consorcioID string) ([]ComunicadoDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(consorcioID)); err != nil {
		return nil, ErrComunicadoInvalid
	}
	rows, err := q.ListComunicados(ctx, cid)
	if err != nil {
		return nil, err
	}
	out := make([]ComunicadoDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, comunicadoDTO(row))
	}
	return out, nil
}

func CreateComunicado(ctx context.Context, q *db.Queries, consorcioID, createdBy string, in ComunicadoInput) (ComunicadoDTO, error) {
	if strings.TrimSpace(in.Titulo) == "" || strings.TrimSpace(in.Cuerpo) == "" {
		return ComunicadoDTO{}, ErrComunicadoInvalid
	}
	var cid, actor pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(consorcioID)); err != nil {
		return ComunicadoDTO{}, ErrComunicadoInvalid
	}
	if err := actor.Scan(strings.TrimSpace(createdBy)); err != nil {
		return ComunicadoDTO{}, ErrComunicadoInvalid
	}
	row, err := q.CreateComunicado(ctx, db.CreateComunicadoParams{
		ConsorcioID:   cid,
		Titulo:        strings.TrimSpace(in.Titulo),
		Cuerpo:        strings.TrimSpace(in.Cuerpo),
		Destinatarios: normalizeDestinatarios(in.Destinatarios),
		CreatedBy:     actor,
	})
	if err != nil {
		return ComunicadoDTO{}, err
	}
	return comunicadoDTO(row), nil
}

func PublicarComunicado(ctx context.Context, q *db.Queries, comunicadoID string) (ComunicadoDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(comunicadoID)); err != nil {
		return ComunicadoDTO{}, ErrComunicadoInvalid
	}
	current, err := q.GetComunicado(ctx, cid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ComunicadoDTO{}, ErrComunicadoNotFound
		}
		return ComunicadoDTO{}, err
	}
	if current.Estado == "publicado" {
		return comunicadoDTO(current), nil
	}
	published, err := q.PublishComunicado(ctx, cid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			refreshed, readErr := q.GetComunicado(ctx, cid)
			if readErr != nil {
				return comunicadoDTO(current), nil
			}
			return comunicadoDTO(refreshed), nil
		}
		return ComunicadoDTO{}, err
	}
	payload, err := json.Marshal(ComunicadoPayload{
		ComunicadoID:  published.ID.String(),
		ConsorcioID:   published.ConsorcioID.String(),
		Titulo:        published.Titulo,
		Cuerpo:        published.Cuerpo,
		Destinatarios: published.Destinatarios,
	})
	if err != nil {
		return ComunicadoDTO{}, err
	}
	if _, err := q.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
		CorrelationID: published.ID,
		EventType:     EventComunicadoPublished,
		Payload:       payload,
	}); err != nil {
		return ComunicadoDTO{}, err
	}
	return comunicadoDTO(published), nil
}

func comunicadoDTO(row db.Comunicado) ComunicadoDTO {
	return ComunicadoDTO{
		ID:            row.ID.String(),
		ConsorcioID:   row.ConsorcioID.String(),
		Titulo:        row.Titulo,
		Cuerpo:        row.Cuerpo,
		Destinatarios: row.Destinatarios,
		Estado:        row.Estado,
		PublicadoAt:   timestamptzPtr(row.PublicadoAt),
		CreatedAt:     timestamptzTime(row.CreatedAt),
		UpdatedAt:     timestamptzTime(row.UpdatedAt),
	}
}

func normalizeDestinatarios(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "todos":
		return "todos"
	default:
		return "unidades"
	}
}

func timestamptzTime(v pgtype.Timestamptz) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time.UTC()
}

func timestamptzPtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time.UTC()
	return &t
}
