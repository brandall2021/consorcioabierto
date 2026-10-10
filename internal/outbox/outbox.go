package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/notificaciones"
	"github.com/brandall2021/consorcioabierto/internal/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EventPublished           = "liquidacion.publicada"
	EventComunicadoPublished = "comunicado.publicado"
)

type MailDriver interface {
	Send(to, subject, body string) error
}

type MockDriver struct{ Log *slog.Logger }

type MailpitDriver struct{ BaseURL string }

type SimplePDFGenerator struct{}

// PDFGenerator genera el PDF de una liquidación publicada.
type PDFGenerator interface {
	Generate(p LiquidacionPayload) ([]byte, error)
}

type Worker struct {
	Log     *slog.Logger
	Pool    *pgxpool.Pool
	Queries *db.Queries
	Mail    MailDriver
	PDFGen  PDFGenerator
}

type LiquidacionPayload struct {
	LiquidacionID string `json:"liquidacion_id"`
	Periodo       string `json:"periodo"`
	ConsorcioID   string `json:"consorcio_id"`
}

type comunicadoPublicacionPayload struct {
	ComunicadoID  string `json:"comunicado_id"`
	ConsorcioID   string `json:"consorcio_id"`
	Titulo        string `json:"titulo"`
	Cuerpo        string `json:"cuerpo"`
	Destinatarios int64  `json:"destinatarios"`
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.Pool == nil || w.Queries == nil {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.drain(ctx)
		}
	}
}

func (w *Worker) drain(ctx context.Context) {
	events, err := w.Queries.ClaimPendingOutboxEvents(ctx, 25)
	if err != nil {
		w.Log.Error("outbox claim", "error", err)
		return
	}
	for _, event := range events {
		if err := w.process(ctx, event); err != nil {
			w.Log.Error("outbox process", "tenant_id", event.TenantID.String(), "event_id", event.ID.String(), "error", err)
		}
	}
}

func (w *Worker) process(ctx context.Context, event db.OutboxEvent) error {
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tenancy.SetTenant(ctx, tx, event.TenantID.String()); err != nil {
		return err
	}
	q := db.New(tx)

	switch event.EventType {
	case EventComunicadoPublished:
		var payload comunicadoPublicacionPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return w.markFailed(ctx, tx, q, event, err)
		}
		// 5.2: al publicar, las notificaciones quedan encoladas; 6.2: el worker
		// las envia. El fan-out es idempotente, asi que un reintento de este
		// evento no duplica avisos.
		res, err := notificaciones.NotificarComunicado(ctx, q, payload.ConsorcioID, payload.ComunicadoID, payload.Titulo, payload.Cuerpo)
		if err != nil {
			return w.markFailed(ctx, tx, q, event, err)
		}
		w.Log.Info("comunicado publicado",
			"tenant_id", event.TenantID.String(),
			"comunicado_id", payload.ComunicadoID,
			"destinatarios", payload.Destinatarios,
			"notificados", res.Destinatarios,
			"notificaciones_creadas", res.Creadas,
			"notificaciones_omitidas", res.Omitidas)
	case EventPublished:
		if err := w.processPublicacion(ctx, q, event); err != nil {
			return w.markFailed(ctx, tx, q, event, err)
		}
	default:
		return w.markFailed(ctx, tx, q, event, fmt.Errorf("evento outbox desconocido: %s", event.EventType))
	}

	if err := q.MarkOutboxEventProcessed(ctx, db.MarkOutboxEventProcessedParams{ID: event.ID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *Worker) processPublicacion(ctx context.Context, q *db.Queries, event db.OutboxEvent) error {
	var payload LiquidacionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("payload inválido: %w", err)
	}

	if w.PDFGen == nil {
		return fmt.Errorf("outbox: PDF generator no configurado")
	}
	pdfBytes, err := w.PDFGen.Generate(payload)
	if err != nil {
		return fmt.Errorf("generar PDF: %w", err)
	}
	w.Log.Info("pdf generado",
		"tenant_id", event.TenantID.String(),
		"liquidacion_id", payload.LiquidacionID,
		"bytes", len(pdfBytes))

	// 5.2: al publicar, las notificaciones quedan encoladas; 6.2: el worker las
	// envia. El fan-out es idempotente, asi que un reintento de este evento no
	// duplica avisos.
	res, err := notificaciones.NotificarLiquidacion(ctx, q, payload.ConsorcioID, payload.LiquidacionID, tituloLiquidacion(payload.Periodo), cuerpoLiquidacion(payload.Periodo))
	if err != nil {
		return fmt.Errorf("notificar liquidación: %w", err)
	}
	w.Log.Info("liquidacion publicada",
		"tenant_id", event.TenantID.String(),
		"liquidacion_id", payload.LiquidacionID,
		"notificados", res.Destinatarios,
		"notificaciones_creadas", res.Creadas,
		"notificaciones_omitidas", res.Omitidas)

	// TODO(H3.6): enviar email a vínculos de UF
	if w.Mail != nil {
		w.Log.Info("email enviado (mock)", "liquidacion_id", payload.LiquidacionID)
	}

	return nil
}

// tituloLiquidacion y cuerpoLiquidacion derivan el texto del aviso del periodo.
// El payload no transporta titulo/cuerpo porque el worker los puede recomponer
// de forma determinista; el texto no depende del cliente.
func tituloLiquidacion(periodo string) string {
	if p := strings.TrimSpace(periodo); p != "" {
		return "Liquidación " + p
	}
	return "Liquidación publicada"
}

func cuerpoLiquidacion(periodo string) string {
	if p := strings.TrimSpace(periodo); p != "" {
		return fmt.Sprintf("Período %s. Ya está disponible en el portal.", p)
	}
	return "Ya está disponible en el portal."
}

func (w *Worker) markFailed(ctx context.Context, tx pgx.Tx, q *db.Queries, event db.OutboxEvent, cause error) error {
	if err := q.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{ID: event.ID, LastError: cause.Error(), NextAttemptAt: pgtype.Timestamptz{Time: time.Now().Add(5 * time.Second), Valid: true}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return cause
}
