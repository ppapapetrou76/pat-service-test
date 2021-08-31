package prometheus

import (
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPMetricer defines how to create HTTPHandlerMetricer.
type HTTPMetricer interface {
	WithHandler(method, handler string) HTTPHandlerMetricer
}

// HTTPHandlerMetricer defines how http metric works.
type HTTPHandlerMetricer interface {
	Inc(statusCode int)
	IncInFlight()
	DecInFlight()

	ObserveRequestSize(sizeInBytes int64)
	ObserveResponseSize(sizeInBytes int64)
	ObserveDuration(duration time.Duration)
}

// NewHTTPMetric returns a new metricer that records HTTP request/ response metrics
//
// Usage:
// m := NewHTTPMetric().WithHandler(method, handler)
// m.IncInFlight()
// defer m.IncInFlight()
// ...
//
func NewHTTPMetric() HTTPMetricer {
	return &httpMetric{}
}

func NewHTTPHandlerMetric(method, handler string) HTTPHandlerMetricer {
	return &httpHandlerMetric{
		prometheus.Labels{
			"method":  strings.ToLower(method),
			"handler": strings.ToLower(handler),
		},
	}
}

const (
	kb, mb = 1e3, 1e6
)

var (
	sizeDefBucketsInByte = []float64{
		100,
		200,
		500,
		kb,
		2 * kb,
		5 * kb,
		10 * kb,
		20 * kb,
		50 * kb,
		100 * kb,
		500 * kb,
		mb,
		2 * mb,
		5 * mb,
		10 * mb,
	}

	httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "The current number of HTTP requests is being served.",
		},
		[]string{"method", "handler"},
	)
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests made and responded.",
		},
		[]string{"method", "handler", "code"},
	)

	httpRequestSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_requests_size_bytes",
			Help:    "The HTTP request sizes in bytes.",
			Buckets: sizeDefBucketsInByte,
		},
		[]string{"method", "handler"},
	)

	httpResponseSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_responses_size_bytes",
			Help:    "The HTTP response sizes in bytes.",
			Buckets: sizeDefBucketsInByte,
		},
		[]string{"method", "handler"},
	)
	httpDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_requests_duration_seconds",
			Help:    "The HTTP request latencies in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "handler"},
	)
)

type httpMetric struct{}

func (m httpMetric) WithHandler(method, handler string) HTTPHandlerMetricer {
	return NewHTTPHandlerMetric(method, handler)
}

type httpHandlerMetric struct {
	labels prometheus.Labels
}

func (m httpHandlerMetric) IncInFlight() {
	httpRequestsInFlight.With(m.labels).Inc()
}

func (m httpHandlerMetric) DecInFlight() {
	httpRequestsInFlight.With(m.labels).Dec()
}

func (m httpHandlerMetric) Inc(statusCode int) {
	httpRequestsTotal.With(prometheus.Labels{
		"method":  m.labels["method"],
		"handler": m.labels["handler"],
		"code":    strconv.FormatInt(int64(statusCode), 10),
	}).Inc()
}

func (m httpHandlerMetric) ObserveRequestSize(sizeInBytes int64) {
	httpRequestSizeBytes.With(m.labels).Observe(float64(sizeInBytes))
}

func (m httpHandlerMetric) ObserveResponseSize(sizeInBytes int64) {
	httpResponseSizeBytes.With(m.labels).Observe(float64(sizeInBytes))
}

func (m httpHandlerMetric) ObserveDuration(duration time.Duration) {
	httpDurationSeconds.With(m.labels).Observe(duration.Seconds())
}
