package prometheus

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// DBCodeOK represents query OK that used for DBQueryMetrics.Processed.
	DBCodeOK = "ok"

	// DBCodeFailed represents query failed that used for DBQueryMetrics.Processed.
	DBCodeFailed = "failed"

	// DBCodeNotFound represents query not found that used for DBQueryMetrics.Processed.
	DBCodeNotFound = "not_found"

	// DBCodeCanceled represents query canceled that used for DBQueryMetrics.Processed.
	DBCodeCanceled = "canceled"
)

var (
	dbQueryInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_queries_in_flight",
			Help: "The current number of database queries is being processed.",
		},
		[]string{"database", "method"},
	)

	dbQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries made and responded.",
		},
		[]string{"database", "method", "code"},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_queries_duration_seconds",
			Help:    "The database query latencies in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"database", "method"},
	)
)

// NewDBQueryMetrics returns a metrics which used for monitoring database query
//
// Usage example:
//
// m := NewDBQueryMetrics("db_name", "get_user_by_device_id")
// m.Begin()
// defer m.End()
//
// u, err := db.GetUserByDeviceID(ctx, "device-id")
// m.Processed(code(ctx, err))
// return u, err.
//
func NewDBQueryMetrics(database, method string) *DBQueryMetrics {
	return &DBQueryMetrics{
		database: database,
		method:   method,

		inFlight:     dbQueryInFlight,
		total:        dbQueryTotal,
		durationSecs: dbQueryDuration,

		t: time.Now(),
	}
}

// DBQueryMetrics represents a collection of metrics to be registered on a Prometheus metrics registry.
type DBQueryMetrics struct {
	database string
	method   string

	inFlight     *prometheus.GaugeVec
	total        *prometheus.CounterVec
	durationSecs *prometheus.HistogramVec

	t     time.Time
	begin bool
}

// Begin the monitoring.
func (m *DBQueryMetrics) Begin() {
	if m.begin {
		return
	}
	m.t = time.Now()
	m.incInFlight()
	m.begin = true
}

// End the monitoring.
func (m *DBQueryMetrics) End() {
	if !m.begin {
		return
	}
	m.decInFlight()
	m.begin = false
}

// Processed logs the result code.
func (m DBQueryMetrics) Processed(code string) {
	m.processed(code)
	m.observeProcessingTime(time.Since(m.t))
}

func (m DBQueryMetrics) processed(code string) {
	label := m.label()
	label["code"] = strings.ToLower(code)

	m.total.With(label).Inc()
}

func (m DBQueryMetrics) incInFlight() {
	m.inFlight.With(m.label()).Inc()
}

func (m DBQueryMetrics) decInFlight() {
	m.inFlight.With(m.label()).Dec()
}

func (m DBQueryMetrics) observeProcessingTime(t time.Duration) {
	m.durationSecs.With(m.label()).Observe(t.Seconds())
}

func (m DBQueryMetrics) label() prometheus.Labels {
	return prometheus.Labels{
		"database": strings.ToLower(m.database),
		"method":   strings.ToLower(m.method),
	}
}
