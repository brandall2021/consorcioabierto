package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceHTTPRequestCreatesSpanWithRouteAndStatus(t *testing.T) {
	prev := otel.GetTracerProvider()
	defer otel.SetTracerProvider(prev)

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	router := http.NewServeMux()
	router.HandleFunc("/portal", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := TraceHTTPRequest(router)
	req := httptest.NewRequest(http.MethodGet, "/portal", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	span := tracetest.SpanStubFromReadOnlySpan(spans[0])
	if span.Name != "GET /portal" {
		t.Fatalf("unexpected span name: %s", span.Name)
	}
	if !hasAttr(span.Attributes, "http.route", "/portal") {
		t.Fatalf("missing route attribute: %+v", span.Attributes)
	}
	if !hasAttr(span.Attributes, "http.status_code", int64(http.StatusNoContent)) {
		t.Fatalf("missing status attribute: %+v", span.Attributes)
	}
}

func hasAttr(attrs []attribute.KeyValue, key string, want any) bool {
	for _, attr := range attrs {
		if string(attr.Key) != key {
			continue
		}
		switch v := want.(type) {
		case string:
			return attr.Value.AsString() == v
		case int64:
			return attr.Value.AsInt64() == v
		}
	}
	return false
}
