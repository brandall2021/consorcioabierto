package observability

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerExposesPrometheusMetrics(t *testing.T) {
	ObserveHTTPRequest("GET", "/portal", 200, 125*time.Millisecond)
	IncLoginFailure()
	IncOutboxProcessed()
	IncOutboxFailed()

	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))

	body := rec.Body.String()
	for _, want := range []string{"http_requests_total", "http_request_duration_seconds", "login_failures_total", "outbox_processed_total", "outbox_failed_total"} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q:\n%s", want, body)
		}
	}
}
