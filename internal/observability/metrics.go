package observability

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsOnce sync.Once
	registry    = prometheus.NewRegistry()

	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total de requests HTTP servidas.",
	}, []string{"method", "route", "status"})
	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duración de requests HTTP en segundos.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
	loginFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "login_failures_total",
		Help: "Total de fallos de login.",
	})
	outboxProcessedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "outbox_processed_total",
		Help: "Total de eventos outbox procesados.",
	})
	outboxFailedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "outbox_failed_total",
		Help: "Total de eventos outbox fallidos.",
	})
)

func ensureMetrics() {
	metricsOnce.Do(func() {
		registry.MustRegister(httpRequestsTotal, httpRequestDuration, loginFailuresTotal, outboxProcessedTotal, outboxFailedTotal)
	})
}

func ObserveHTTPRequest(method, route string, status int, duration time.Duration) {
	ensureMetrics()
	route = normalizeRoute(route)
	method = normalizeLabel(method, "UNKNOWN")
	statusLabel := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(method, route, statusLabel).Inc()
	httpRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

func IncLoginFailure() {
	ensureMetrics()
	loginFailuresTotal.Inc()
}

func IncOutboxProcessed() {
	ensureMetrics()
	outboxProcessedTotal.Inc()
}

func IncOutboxFailed() {
	ensureMetrics()
	outboxFailedTotal.Inc()
}

func Handler() http.HandlerFunc {
	ensureMetrics()
	return func(w http.ResponseWriter, r *http.Request) {
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

func normalizeRoute(route string) string {
	if route == "" {
		return "unknown"
	}
	return route
}

func normalizeLabel(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
