package observability

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracing configura un tracer provider OTel con exportación stdout,
// suficiente para desarrollo y verificación local sin collector externo.
func InitTracing(serviceName, environment string) (func(context.Context) error, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	res, err := sdkresource.New(context.Background(),
		sdkresource.WithSchemaURL(semconv.SchemaURL),
		sdkresource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(environment),
		),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp.Shutdown, nil
}

// TraceHTTPRequest crea un span por request HTTP y anota método, ruta y status.
func TraceHTTPRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		route := RoutePattern(r)
		tracer := otel.Tracer("github.com/brandall2021/consorcioabierto/internal/observability/http")
		ctx, span := tracer.Start(r.Context(), spanName(r.Method, route))
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r.WithContext(ctx))
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", route),
			attribute.Int("http.status_code", status),
			attribute.Float64("http.duration_seconds", time.Since(start).Seconds()),
		)
		if status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(status))
		}
		span.End()
	})
}

// RoutePattern devuelve el patrón de ruta chi cuando existe, o la URL del request.
func RoutePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := strings.TrimSpace(rc.RoutePattern()); pattern != "" {
			return pattern
		}
	}
	if r != nil && r.URL != nil && r.URL.Path != "" {
		return r.URL.Path
	}
	return "unknown"
}

func spanName(method, route string) string {
	method = strings.TrimSpace(method)
	if method == "" {
		method = "HTTP"
	}
	route = normalizeRoute(route)
	return method + " " + route
}
