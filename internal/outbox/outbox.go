// Outbox worker: goroutine que pollorea outbox_events pendientes (§5.4). Cada
// evento se procesa en su propia transacción con el contexto RLS del tenant del
// evento (los Mark* SQL filtran por app.current_tenant_id()).
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/observability"
	"github.com/brandall2021/consorcioabierto/internal/tenancy"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	maxRetries               = 5
	pollInterval             = 2 * time.Second
	batchSize                = 10
	EventPublished           = "liquidacion.publicada"
	EventComunicadoPublished = "comunicado.publicado"
)

var _ = maxRetries

// PDFGenerator genera el PDF de una liquidación publicada.
type PDFGenerator interface {
	Generate(liquidacion LiquidacionPayload) ([]byte, error)
}

type LiquidacionPayload struct {
	LiquidacionID string `json:"liquidacion_id"`
	Periodo       string `json:"periodo"`
	ConsorcioID   string `json:"consorcio_id"`
}

type ComunicadoPayload struct {
	ComunicadoID  string `json:"comunicado_id"`
	ConsorcioID   string `json:"consorcio_id"`
	Titulo        string `json:"titulo"`
	Cuerpo        string `json:"cuerpo"`
	Destinatarios string `json:"destinatarios"`
}

// Worker procesa eventos outbox pendientes.
type Worker struct {
	Log    *slog.Logger
	Pool   *pgxpool.Pool
	Mail   MailDriver
	PDFGen PDFGenerator
}

// Run inicia el worker loop. Bloquea hasta que ctx se cancele.
func (w *Worker) Run(ctx context.Context) {
	w.Log.Info("outbox worker iniciado")
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Log.Info("outbox worker detenido")
			return
		case <-ticker.C:
			w.processBatch()
		}
	}
}

func (w *Worker) processBatch() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tracer := otel.Tracer("github.com/brandall2021/consorcioabierto/internal/outbox")
	ctx, span := tracer.Start(ctx, "outbox.process_batch")
	defer span.End()

	events, err := db.New(w.Pool).GetPendingOutboxEvents(ctx, int32(batchSize))
	if err != nil {
		span.RecordError(err)
		w.Log.Error("outbox poll", "error", err)
		return
	}
	span.SetAttributes(attribute.Int("outbox.pending_events", len(events)))

	for _, ev := range events {
		if err := w.processEvent(ctx, ev); err != nil {
			w.Log.Error("outbox process", "event_id", ev.ID.String(), "error", err)
			w.markFailed(ev)
			observability.IncOutboxFailed()
		} else {
			observability.IncOutboxProcessed()
		}
	}
}

// processEvent procesa un evento en una transacción con el tenant del evento,
// y la deja marcada 'procesado' si todo sale bien. Un event_type desconocido
// tampoco debe trabar la cola: se marca procesado y se loguea.
func (w *Worker) processEvent(ctx context.Context, ev db.OutboxEvent) error {
	tracer := otel.Tracer("github.com/brandall2021/consorcioabierto/internal/outbox")
	ctx, span := tracer.Start(ctx, "outbox.process_event")
	defer span.End()
	span.SetAttributes(
		attribute.String("outbox.event_id", ev.ID.String()),
		attribute.String("outbox.event_type", ev.EventType),
		attribute.String("outbox.tenant_id", ev.TenantID.String()),
	)
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("begin tx: %w", err)
	}
	// El rollback usa Background: si el ctx de procesamiento expiró, igual debe
	// liberarse la transacción/conn del pool.
	defer func() { _ = tx.Rollback(context.Background()) }()

	if err := tenancy.SetTenant(ctx, tx, ev.TenantID.String()); err != nil {
		span.RecordError(err)
		return fmt.Errorf("set tenant: %w", err)
	}
	q := db.New(tx)

	switch ev.EventType {
	case EventPublished:
		if err := w.processPublicacion(ctx, q, ev); err != nil {
			return err
		}
	case EventComunicadoPublished:
		if err := w.processComunicado(ctx, q, ev); err != nil {
			return err
		}
	default:
		w.Log.Warn("outbox event_type desconocido", "type", ev.EventType)
		span.AddEvent("unknown_event_type")
	}

	if err := q.MarkOutboxProcessed(ctx, ev.ID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("marcar procesado: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (w *Worker) processPublicacion(ctx context.Context, q *db.Queries, ev db.OutboxEvent) error {
	tracer := otel.Tracer("github.com/brandall2021/consorcioabierto/internal/outbox")
	_, span := tracer.Start(ctx, "outbox.process_publicacion")
	defer span.End()
	var payload LiquidacionPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		span.RecordError(err)
		return fmt.Errorf("payload inválido: %w", err)
	}

	if w.PDFGen == nil {
		span.AddEvent("pdf_generator_missing")
		return fmt.Errorf("outbox: PDF generator no configurado")
	}
	pdfBytes, err := w.PDFGen.Generate(payload)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("generar PDF: %w", err)
	}
	w.Log.Info("PDF generado", "liquidacion_id", payload.LiquidacionID, "bytes", len(pdfBytes))

	// TODO(H3.6): enviar email a vínculos de UF
	if w.Mail != nil {
		w.Log.Info("email enviado (mock)", "liquidacion_id", payload.LiquidacionID)
	}

	return nil
}

func (w *Worker) processComunicado(ctx context.Context, q *db.Queries, ev db.OutboxEvent) error {
	tracer := otel.Tracer("github.com/brandall2021/consorcioabierto/internal/outbox")
	ctx, span := tracer.Start(ctx, "outbox.process_comunicado")
	defer span.End()
	var payload ComunicadoPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		span.RecordError(err)
		return fmt.Errorf("payload inválido: %w", err)
	}

	recipients, err := q.ListComunicadoRecipients(ctx, ev.CorrelationID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("listar destinatarios: %w", err)
	}
	span.SetAttributes(attribute.Int("outbox.recipients", len(recipients)))
	if len(recipients) == 0 {
		w.Log.Info("comunicado sin destinatarios", "comunicado_id", payload.ComunicadoID)
		return nil
	}
	if w.Mail == nil {
		w.Log.Info("comunicado sin mailer", "comunicado_id", payload.ComunicadoID, "destinatarios", len(recipients))
		return nil
	}
	for _, rcpt := range recipients {
		if !rcpt.Email.Valid || rcpt.Email.String == "" {
			continue
		}
		body := payload.Cuerpo + "\n\n--\nConsorcioAbierto"
		if err := w.Mail.Send(rcpt.Email.String, payload.Titulo, body); err != nil {
			span.RecordError(err)
			return fmt.Errorf("send comunicado: %w", err)
		}
	}
	w.Log.Info("comunicado enviado", "comunicado_id", payload.ComunicadoID, "destinatarios", len(recipients))
	return nil
}

// markFailed marca el evento 'fallido' en una transacción nueva con el tenant
// del evento. MarkOutboxFailed fija estado='fallido' de forma inmediata
// (terminal): el worker no reintenta en vivo; intentos + next_attempt_at
// (backoff exponencial) quedan registrados como diario diagnóstico y apoyo
// para un re-encolado manual.
func (w *Worker) markFailed(ev db.OutboxEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tracer := otel.Tracer("github.com/brandall2021/consorcioabierto/internal/outbox")
	ctx, span := tracer.Start(ctx, "outbox.mark_failed")
	defer span.End()
	span.SetAttributes(attribute.String("outbox.event_id", ev.ID.String()))

	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		w.Log.Error("outbox mark failed begin", "event_id", ev.ID.String(), "error", err)
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if err := tenancy.SetTenant(ctx, tx, ev.TenantID.String()); err != nil {
		span.RecordError(err)
		w.Log.Error("outbox mark failed tenant", "event_id", ev.ID.String(), "error", err)
		return
	}
	if err := db.New(tx).MarkOutboxFailed(ctx, ev.ID); err != nil {
		span.RecordError(err)
		w.Log.Error("outbox mark failed", "event_id", ev.ID.String(), "error", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		w.Log.Error("outbox mark failed commit", "event_id", ev.ID.String(), "error", err)
	}
}
