package metrics

import (
	"net/http"
	"time"

	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type Middleware struct {
	RequestsDuration prometheus.Histogram
	RequestsCurrent  prometheus.Gauge
	RequestsStatus   *prometheus.CounterVec
	ClientErrors     prometheus.Counter
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func NewMiddleware() Middleware {
	return Middleware{}
}

func (m Middleware) Handler(namespace string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.RequestsDuration = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "The duration of the requests to " + namespace,
			},
		)

		m.RequestsCurrent = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "requests_current",
				Help:      "The current number of requests to " + namespace,
			},
		)

		m.RequestsStatus = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "The total number of requests to the " + namespace + " by status, method and path.",
			},
			[]string{"code", "method", "path"},
		)

		m.ClientErrors = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors",
				Help:      "The total number of " + namespace + " client errors",
			})

		start := time.Now()
		m.RequestsCurrent.Inc()

		lrw := newLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		statusCode := lrw.statusCode

		m.RequestsStatus.WithLabelValues(strconv.Itoa(statusCode), r.Method, r.URL.Path).Inc()

		if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
			m.ClientErrors.Inc()
		}

		m.RequestsCurrent.Dec()
		m.RequestsDuration.Observe(float64(time.Since(start).Seconds()))
	})
}
