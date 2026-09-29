// Package observability concentra trazas (OpenTelemetry) y métricas
// (Prometheus) de todos los procesos ([ADR-0011], §10.3).
package observability

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Registry es el registry Prometheus que expone `Handler` en /metrics.
var reg = prometheus.NewRegistry()

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total de requests HTTP por ruta, método y status.",
	}, []string{"route", "method", "status"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duración de requests HTTP por ruta y status.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route", "status"})

	httpRequestsInFlight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Requests HTTP en curso por ruta.",
	}, []string{"route"})

	loginFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "auth_login_failures_total",
		Help: "Intentos de login fallidos (registrados en auto.failed_login).",
	})

	databasePoolConns = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "database_pool_connections",
		Help: "Conexiones del pool pgx por estado.",
	}, []string{"state"})

	databasePoolMax = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "database_pool_max_connections",
		Help: "Máximo de conexiones del pool pgx.",
	})

	outboxEventsByState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "outbox_events_by_state",
		Help: "Eventos del outbox por estado (app.outbox_metrics).",
	}, []string{"estado"})
)

func init() {
	reg.MustRegister(
		httpRequestsTotal, httpRequestDuration, httpRequestsInFlight, loginFailures,
		databasePoolConns, databasePoolMax, outboxEventsByState,
	)
}

// tracerFor usa siempre el provider global activo (noop si no hay InitTracing).
func tracerFor() trace.Tracer { return otel.Tracer("consorcioabierto") }

// InitTracing instala el TracerProvider global y devuelve su shutdown.
// exporter: console (por defecto), otlp (requiere endpoint) o none.
func InitTracing(service, env, exporter, endpoint string) (func(context.Context) error, error) {
	if exporter == "" {
		exporter = "console"
	}
	if exporter == "none" {
		return func(context.Context) error { return nil }, nil
	}

	res := resource.NewSchemaless(
		attribute.String("service.name", service),
		attribute.String("deployment.environment", env),
	)

	var spanExporter sdktrace.SpanExporter
	switch exporter {
	case "otlp":
		opts := []otlptracehttp.Option{}
		if endpoint != "" {
			opts = append(opts, otlptracehttp.WithEndpointURL(endpoint))
		}
		exp, err := otlptracehttp.New(context.Background(), opts...)
		if err != nil {
			return nil, err
		}
		spanExporter = exp
	default:
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		spanExporter = exp
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// TraceHTTPRequest crea un span OTel por request y emite las métricas HTTP.
func TraceHTTPRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := RoutePattern(r)
		ctx, span := tracerFor().Start(r.Context(), route, trace.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("url.path", r.URL.Path),
			attribute.String("request.id", middleware.GetReqID(r.Context())),
		))
		r = r.WithContext(ctx)

		httpRequestsInFlight.WithLabelValues(route).Inc()
		defer httpRequestsInFlight.WithLabelValues(route).Dec()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		label := strconv.Itoa(status)
		seconds := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(route, r.Method, label).Inc()
		httpRequestDuration.WithLabelValues(route, label).Observe(seconds)

		span.SetAttributes(attribute.Int("http.response.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, "http status >= 500")
		}
		span.End()
	})
}

// Handler sirve las métricas Prometheus en /metrics.
func Handler() http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

func IncLoginFailure() { loginFailures.Inc() }

// RoutePattern devuelve el patrón de ruta chi y, si no hay contexto de ruta,
// la ruta real (mantiene acotadas las etiquetas de métricas).
func RoutePattern(r *http.Request) string {
	if ctx := chi.RouteContext(r.Context()); ctx != nil {
		if p := ctx.RoutePattern(); p != "" {
			return p
		}
	}
	return r.URL.Path
}

// RegisterDBPool publica gauges del estado del pool pgx cada 15s.
func RegisterDBPool(ctx context.Context, pool *pgxpool.Pool) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				st := pool.Stat()
				databasePoolMax.Set(float64(st.MaxConns()))
				databasePoolConns.WithLabelValues("total").Set(float64(st.TotalConns()))
				databasePoolConns.WithLabelValues("idle").Set(float64(st.IdleConns()))
				databasePoolConns.WithLabelValues("acquired").Set(float64(st.AcquiredConns()))
			}
		}
	}()
}

// RegisterOutbox publica gauges del outbox (app.outbox_metrics) cada 15s.
// app.outbox_metrics es SECURITY DEFINER y ve filas de todos los tenants.
func RegisterOutbox(ctx context.Context, pool *pgxpool.Pool) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collectOutbox(ctx, pool)
			}
		}
	}()
}

func collectOutbox(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `SELECT estado, total FROM app.outbox_metrics()`)
	if err != nil {
		slog.Default().Warn("outbox metrics", "error", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var estado string
		var total int64
		if err := rows.Scan(&estado, &total); err != nil {
			slog.Default().Warn("outbox metrics", "error", err)
			return
		}
		outboxEventsByState.WithLabelValues(estado).Set(float64(total))
	}
	if err := rows.Err(); err != nil {
		slog.Default().Warn("outbox metrics", "error", err)
	}
}
