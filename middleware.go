package metrics

import (
	"net/http"
	"time"

	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

var (
	RequestsDuration prometheus.Histogram
	RequestsCurrent  prometheus.Gauge
	RequestsStatus   *prometheus.CounterVec
	ClientErrors     prometheus.Counter
)

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func Handler(namespace string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RequestsDuration = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "The duration of the requests to the Statistics service.",
			},
		)

		RequestsCurrent = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "requests_current",
				Help:      "The current number of requests to " + namespace,
			},
		)

		RequestsStatus = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "The total number of requests to the " + namespace + " by status, method and path.",
			},
			[]string{"code", "method", "path"},
		)

		ClientErrors = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors",
				Help:      "The total number of " + namespace + " client errors",
			})

		start := time.Now()
		RequestsCurrent.Inc()

		lrw := newLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		statusCode := lrw.statusCode

		RequestsStatus.WithLabelValues(strconv.Itoa(statusCode), r.Method, r.URL.Path).Inc()

		if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
			ClientErrors.Inc()
		}

		RequestsCurrent.Dec()
		RequestsDuration.Observe(float64(time.Since(start).Seconds()))
	})
}
