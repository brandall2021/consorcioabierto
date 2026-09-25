package server

import (
	"log/slog"
	"net/http"
	"time"

	httpx "github.com/brandall2021/consorcioabierto/apps/api/transport/http"
	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/cobranzas"
	"github.com/brandall2021/consorcioabierto/internal/documentos"
	"github.com/brandall2021/consorcioabierto/internal/identity"
	"github.com/brandall2021/consorcioabierto/internal/observability"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

// New crea el router HTTP raíz con middleware, rutas y handlers.
func New(log *slog.Logger, env string, identityManager *identity.AuthManager, auditRecorder *audit.Recorder, docsEnv documentos.DocsEnv, psp cobranzas.PSP) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	// middleware.RealIP deprecated y removido
	// r.Use(middleware.RealIP)

	r.Use(middleware.Recoverer)
	r.Use(observability.TraceHTTPRequest)
	r.Use(requestLogger(log))
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/metrics", observability.Handler())

	api := chi.NewRouter()
	api.Get("/health", handleHealth)

	h := &httpx.AuthHandlers{Manager: identityManager, Audit: auditRecorder, Docs: docsEnv, PSP: psp}
	httpx.RegisterAuthRoutes(api, h)

	r.Mount("/api/v1", api)

	return r
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			log.Info("http",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"route", observability.RoutePattern(r),
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
				"trace_id", trace.SpanContextFromContext(r.Context()).TraceID().String(),
				"span_id", trace.SpanContextFromContext(r.Context()).SpanID().String(),
			)
		})
	}
}
