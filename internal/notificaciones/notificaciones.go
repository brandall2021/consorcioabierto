// Package notificaciones entrega avisos in-app al consorcista.
//
// La spec pide que al publicar se encolen notificaciones (5.2) y que el worker
// las envie (6.2). El fan-out lo hace el worker del outbox al procesar
// "comunicado.publicado": la escritura es del servicio, la lectura es del
// consorcista.
//
// Reglas:
//   - El destinatario nunca se acepta del cliente: se resuelve por
//     user -> personas -> unidad_personas -> unidades del consorcio, con
//     vinculo vigente. El mismo camino que acota el portal.
//   - La creacion es idempotente. El worker reintenta los eventos fallidos y
//     sin el indice unico de 00029 cada reintento duplicaria el aviso.
//   - Solo el destinatario lee y marca leida; lo garantiza la policy de RLS.
package notificaciones

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
)

const (
	// TipoComunicado es el aviso que genera publicar un comunicado.
	TipoComunicado = "comunicado"
	// RecursoComunicado identifica el recurso al que apunta el aviso.
	RecursoComunicado = "comunicado"
	// limitePorDefecto acota el listado del portal. El consorcista no necesita
	// la tabla completa y el parametro viene del handler, no del cliente.
	limitePorDefecto = 50
	// limiteMaximo impide que un limite enorme convierta el listado en un dump.
	limiteMaximo = 200
	// timeLayout es RFC3339 en UTC, el formato que declara el contrato.
	timeLayout = time.RFC3339
)

var (
	// ErrNotificacionNotFound no distingue "de otro usuario" de inexistente.
	ErrNotificacionNotFound = errors.New("notificación no encontrada")
	// ErrLimiteInvalido cubre un limite negativo o no numerico.
	ErrLimiteInvalido = errors.New("límite inválido")
)

// Queryer agrupa las consultas del paquete para probarlo con dobles.
type Queryer interface {
	ListDestinatariosNotificacion(context.Context, pgtype.UUID) ([]pgtype.UUID, error)
	InsertNotificacion(context.Context, db.InsertNotificacionParams) (db.Notificacion, error)
	ListNotificacionesForCurrentUser(context.Context, db.ListNotificacionesForCurrentUserParams) ([]db.Notificacion, error)
	MarcarNotificacionLeida(context.Context, pgtype.UUID) (db.Notificacion, error)
	CountNotificacionesNoLeidas(context.Context) (int64, error)
}

// DTO es una notificacion del usuario actual (contrato HTTP).
type DTO struct {
	ID          string  `json:"id"`
	Tipo        string  `json:"tipo"`
	Titulo      string  `json:"titulo"`
	Cuerpo      string  `json:"cuerpo"`
	RecursoType string  `json:"recurso_type"`
	RecursoID   *string `json:"recurso_id"`
	LeidaAt     *string `json:"leida_at"`
	CreatedAt   string  `json:"created_at"`
}

// Listado son las notificaciones del usuario actual con el conteo de pendientes.
type Listado struct {
	Data     []DTO `json:"data"`
	NoLeidas int64 `json:"no_leidas"`
}

// Resultado del fan-out de una publicacion.
type Resultado struct {
	// Destinatarios son los usuarios con vinculo vigente al consorcio.
	Destinatarios int `json:"destinatarios"`
	// Creadas son las notificaciones nuevas. Un reintento del worker da 0.
	Creadas int `json:"creadas"`
	// Omitidas son las que ya existian (indice unico de 00029).
	Omitidas int `json:"omitidas"`
}

// NotificarComunicado crea el aviso de un comunicado publicado para cada
// usuario con vinculo vigente a alguna UF del consorcio.
//
// Idempotente por (tenant, usuario, tipo, recurso): si el worker reintenta el
// evento, las notificaciones ya creadas no se duplican y Creadas queda en 0.
// Devuelve error solo ante un fallo real de base; un duplicado no es un fallo.
func NotificarComunicado(ctx context.Context, q Queryer, consorcioID, comunicadoID, titulo, cuerpo string) (Resultado, error) {
	consorcio, err := uuidToPG(consorcioID)
	if err != nil {
		return Resultado{}, err
	}
	recurso, err := uuidToPG(comunicadoID)
	if err != nil {
		return Resultado{}, err
	}
	titulo = strings.TrimSpace(titulo)
	if titulo == "" {
		return Resultado{}, fmt.Errorf("%w: título vacío", ErrLimiteInvalido)
	}

	destinatarios, err := q.ListDestinatariosNotificacion(ctx, consorcio)
	if err != nil {
		return Resultado{}, err
	}
	res := Resultado{Destinatarios: len(destinatarios)}
	for _, usuario := range destinatarios {
		_, err := q.InsertNotificacion(ctx, db.InsertNotificacionParams{
			UserID:      usuario,
			Tipo:        TipoComunicado,
			Titulo:      titulo,
			Cuerpo:      cuerpo,
			RecursoType: RecursoComunicado,
			RecursoID:   recurso,
		})
		switch {
		case err == nil:
			res.Creadas++
		case errors.Is(err, pgx.ErrNoRows):
			// ON CONFLICT DO NOTHING sin RETURNING: el aviso ya existia.
			res.Omitidas++
		default:
			return res, err
		}
	}
	return res, nil
}

// Listar devuelve las notificaciones del usuario actual. El scope no viene del
// cliente: la consulta filtra por app.current_user_id().
func Listar(ctx context.Context, q Queryer, soloNoLeidas bool, limite int) (Listado, error) {
	if limite <= 0 {
		limite = limitePorDefecto
	}
	if limite > limiteMaximo {
		limite = limiteMaximo
	}
	rows, err := q.ListNotificacionesForCurrentUser(ctx, db.ListNotificacionesForCurrentUserParams{
		SoloNoLeidas: soloNoLeidas,
		Limite:       int32(limite),
	})
	if err != nil {
		return Listado{}, err
	}
	noLeidas, err := q.CountNotificacionesNoLeidas(ctx)
	if err != nil {
		return Listado{}, err
	}
	out := make([]DTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDTO(r))
	}
	return Listado{Data: out, NoLeidas: noLeidas}, nil
}

// MarcarLeida marca una notificacion como leida. Si ya estaba leida devuelve la
// misma notificacion sin error, para que el portal pueda repetir el click.
func MarcarLeida(ctx context.Context, q Queryer, id string) (DTO, error) {
	target, err := uuidToPG(id)
	if err != nil {
		return DTO{}, err
	}
	row, err := q.MarcarNotificacionLeida(ctx, target)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Otra notificacion del usuario, o de otro usuario, o ya leida.
			// Los tres casos son indistinguibles para el que pregunta.
			return DTO{}, ErrNotificacionNotFound
		}
		return DTO{}, err
	}
	return toDTO(row), nil
}

func toDTO(n db.Notificacion) DTO {
	dto := DTO{
		ID:          n.ID.String(),
		Tipo:        n.Tipo,
		Titulo:      n.Titulo,
		Cuerpo:      n.Cuerpo,
		RecursoType: n.RecursoType,
		CreatedAt:   n.CreatedAt.Time.UTC().Format(timeLayout),
	}
	if n.RecursoID.Valid {
		s := n.RecursoID.String()
		dto.RecursoID = &s
	}
	if n.LeidaAt.Valid {
		s := n.LeidaAt.Time.UTC().Format(timeLayout)
		dto.LeidaAt = &s
	}
	return dto
}

func uuidToPG(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: %s", ErrLimiteInvalido, s)
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}
