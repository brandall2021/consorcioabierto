package comunicaciones

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const publishScope = "comunicaciones.publish"

var ErrIdempotencyConflict = errors.New("Idempotency-Key ya utilizada con otro request")

type Input struct {
	Titulo        string `json:"titulo"`
	Cuerpo        string `json:"cuerpo"`
	Destinatarios string `json:"destinatarios"`
}

type DTO struct {
	ID            string `json:"id"`
	Titulo        string `json:"titulo"`
	Estado        string `json:"estado"`
	Destinatarios int64  `json:"destinatarios"`
}

type Queryer interface {
	ListUnidades(context.Context, db.ListUnidadesParams) ([]db.Unidade, error)
	ListComunicados(context.Context, db.ListComunicadosParams) ([]db.Comunicado, error)
	GetComunicado(context.Context, pgtype.UUID) (db.Comunicado, error)
	CreateComunicado(context.Context, db.CreateComunicadoParams) (db.Comunicado, error)
	PublishComunicado(context.Context, db.PublishComunicadoParams) (db.Comunicado, error)
	InsertOutboxEvent(context.Context, db.InsertOutboxEventParams) (db.OutboxEvent, error)
	GetIdempotencyKey(context.Context, db.GetIdempotencyKeyParams) (db.IdempotencyKey, error)
	InsertIdempotencyKey(context.Context, db.InsertIdempotencyKeyParams) (int64, error)
	UpdateIdempotencyKey(context.Context, db.UpdateIdempotencyKeyParams) error
}

func List(ctx context.Context, q Queryer, consorcioID string) ([]DTO, error) {
	rows, err := q.ListComunicados(ctx, db.ListComunicadosParams{ConsorcioID: pgtype.UUID{Bytes: uuid.MustParse(consorcioID), Valid: true}})
	if err != nil {
		return nil, err
	}
	items := make([]DTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDTO(row))
	}
	return items, nil
}

func Create(ctx context.Context, q Queryer, consorcioID string, in Input) (DTO, error) {
	if strings.TrimSpace(in.Titulo) == "" || strings.TrimSpace(in.Cuerpo) == "" {
		return DTO{}, errors.New("titulo y cuerpo son obligatorios")
	}
	if in.Destinatarios == "" {
		in.Destinatarios = "todos"
	}
	count, err := destinatariosCount(ctx, q, consorcioID)
	if err != nil {
		return DTO{}, err
	}
	row, err := q.CreateComunicado(ctx, db.CreateComunicadoParams{
		ConsorcioID:        pgtype.UUID{Bytes: uuid.MustParse(consorcioID), Valid: true},
		Titulo:             strings.TrimSpace(in.Titulo),
		Cuerpo:             strings.TrimSpace(in.Cuerpo),
		DestinatariosScope: in.Destinatarios,
		Destinatarios:      count,
	})
	if err != nil {
		return DTO{}, err
	}
	return toDTO(row), nil
}

func Publish(ctx context.Context, q Queryer, comunicadoID, idemKey string) (DTO, error) {
	if strings.TrimSpace(idemKey) == "" {
		return DTO{}, errors.New("Idempotency-Key requerida")
	}
	hash := requestHash(comunicadoID, idemKey)
	inserted, err := q.InsertIdempotencyKey(ctx, db.InsertIdempotencyKeyParams{
		IdempotencyKey: idemKey,
		Scope:          publishScope,
		RequestHash:    hash,
		ResponseJson:   []byte("{}"),
	})
	if err != nil {
		return DTO{}, err
	}
	if inserted == 0 {
		existing, err := q.GetIdempotencyKey(ctx, db.GetIdempotencyKeyParams{Scope: publishScope, IdempotencyKey: idemKey})
		if err != nil {
			return DTO{}, err
		}
		if existing.RequestHash != hash {
			return DTO{}, ErrIdempotencyConflict
		}
		if len(existing.ResponseJson) > 0 && string(existing.ResponseJson) != "{}" {
			var cached DTO
			if err := json.Unmarshal(existing.ResponseJson, &cached); err != nil {
				return DTO{}, err
			}
			return cached, nil
		}
	}

	row, err := q.GetComunicado(ctx, pgtype.UUID{Bytes: uuid.MustParse(comunicadoID), Valid: true})
	if err != nil {
		return DTO{}, err
	}
	if row.Estado != "publicado" {
		row, err = q.PublishComunicado(ctx, db.PublishComunicadoParams{ID: pgtype.UUID{Bytes: uuid.MustParse(comunicadoID), Valid: true}})
		if err != nil {
			return DTO{}, err
		}
		payload, _ := json.Marshal(map[string]any{
			"comunicado_id": row.ID.String(),
			"consorcio_id":  row.ConsorcioID.String(),
			"titulo":        row.Titulo,
			"destinatarios": row.Destinatarios,
		})
		if _, err := q.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
			CorrelationID: row.ID.String(),
			EventType:     "comunicado.publicado",
			Payload:       payload,
		}); err != nil {
			return DTO{}, err
		}
	}
	dto := toDTO(row)
	resp, err := json.Marshal(dto)
	if err != nil {
		return DTO{}, err
	}
	if err := q.UpdateIdempotencyKey(ctx, db.UpdateIdempotencyKeyParams{
		Scope:          publishScope,
		IdempotencyKey: idemKey,
		ResponseJson:   resp,
	}); err != nil {
		return DTO{}, err
	}
	return dto, nil
}

func toDTO(row db.Comunicado) DTO {
	return DTO{
		ID:            row.ID.String(),
		Titulo:        row.Titulo,
		Estado:        row.Estado,
		Destinatarios: row.Destinatarios,
	}
}

func destinatariosCount(ctx context.Context, q Queryer, consorcioID string) (int64, error) {
	units, err := q.ListUnidades(ctx, db.ListUnidadesParams{ConsorcioID: pgtype.UUID{Bytes: uuid.MustParse(consorcioID), Valid: true}, Estado: ""})
	if err != nil {
		return 0, err
	}
	var count int64
	for _, unit := range units {
		if strings.EqualFold(unit.Estado, "activa") {
			count++
		}
	}
	return count, nil
}

func requestHash(comunicadoID, idemKey string) string {
	return fmt.Sprintf("%s|%s", strings.TrimSpace(comunicadoID), strings.TrimSpace(idemKey))
}
