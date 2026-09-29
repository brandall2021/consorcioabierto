package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// resetForTest limpia el registry compartido para que los tests sean
// independientes entre sí. Solo se usa desde pruebas.
func resetForTest() {
	reg = prometheus.NewRegistry()
	reg.MustRegister(
		httpRequestsTotal, httpRequestDuration, httpRequestsInFlight, loginFailures,
		databasePoolConns, databasePoolMax, outboxEventsByState,
	)
}

func TestRoutePatternFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/sin/ruta", nil)
	req = req.WithContext(context.Background())
	if got := RoutePattern(req); got != "/sin/ruta" {
		t.Fatalf("RoutePattern = %q, se esperaba la ruta literal", got)
	}
}

func TestHTTPMetricsSeEmiten(t *testing.T) {
	resetForTest()

	handler := TraceHTTPRequest(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/pruebas", nil))

	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	found := false
	for _, mf := range metrics {
		if mf.GetName() != "http_requests_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if m.GetCounter().GetValue() > 0 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("http_requests_total debe incrementarse con la métrica de la request")
	}
}

func TestIncLoginFailureContador(t *testing.T) {
	resetForTest()

	IncLoginFailure()
	IncLoginFailure()

	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range metrics {
		if mf.GetName() != "auth_login_failures_total" {
			continue
		}
		if got := mf.GetMetric()[0].GetCounter().GetValue(); got != 2 {
			t.Fatalf("login failures = %v, se esperaba 2", got)
		}
		return
	}
	t.Fatal("auth_login_failures_total no está registrada")
}

func TestHandlerExponeTextoPrometheus(t *testing.T) {
	resetForTest()
	outboxEventsByState.WithLabelValues("pendiente").Set(0)

	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"http_requests_total", "auth_login_failures_total", "outbox_events_by_state"} {
		if !strings.Contains(body, want) {
			t.Errorf("/metrics no expone %q", want)
		}
	}
}
