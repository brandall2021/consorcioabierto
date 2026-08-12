package server

import (
	"log/slog"
	"net/http"
	"time"

	httpx "github.com/brandall2021/consorcioabierto/apps/api/transport/http"
	"github.com/brandall2021/consorcioabierto/internal/identity"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// New crea el router HTTP raíz con middleware, rutas y handlers.
func New(log *slog.Logger, env string, identityManager *identity.AuthManager) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	// middleware.RealIP deprecated y removido
	// r.Use(middleware.RealIP)

	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	api := chi.NewRouter()
	api.Get("/health", handleHealth)

	h := &httpx.AuthHandlers{Manager: identityManager}
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
			log.Info("http",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
