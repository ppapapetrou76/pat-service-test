package prometheus

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	logLevels = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "logrus",
			Name:      "logs_total",
			Help:      "Total number of logs recorded in the system.",
		},
		[]string{"level"},
	)

	httpLogLevels = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_logrus_logs_total",
			Help: "Total number of logs recorded in http service.",
		},
		[]string{"level", "method", "handler"},
	)

	grpcLogLevels = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_logrus_logs_total",
			Help: "Total number of logs recorded in grpc services.",
		},
		[]string{"level", "grpc_service", "grpc_method"},
	)
)

// NewLogsMetrics returns a metrics which used for monitoring logrus.
func NewLogsMetrics() *LogsMetrics {
	return &LogsMetrics{
		logLevels:     logLevels,
		httpLogLevels: httpLogLevels,
		grpcLogLevels: grpcLogLevels,
	}
}

// LogsMetrics represents a collection of metrics to be registered on a Logrus metrics registry.
type LogsMetrics struct {
	logLevels     *prometheus.CounterVec
	httpLogLevels *prometheus.CounterVec
	grpcLogLevels *prometheus.CounterVec
}

// Hook returns a logrus hook that could registers into logrus system
//
// Example:
// m := NewLogsMetrics()
// l := logrus.New()
// l.AddHook().Hook(m.Hook()).
func (m *LogsMetrics) Hook() logrus.Hook {
	return &logrusHook{
		reporter: &logrusMetricReporter{m},
	}
}

var defaultLogrusHookLevels = []logrus.Level{
	logrus.DebugLevel,
	logrus.InfoLevel,
	logrus.WarnLevel,
	logrus.ErrorLevel,
	logrus.FatalLevel,
	logrus.PanicLevel,
}

type logrusReporter interface {
	Inc(lv logrus.Level)
	IncHTTP(lv logrus.Level, method, handler string)
	IncRPC(lv logrus.Level, grpcService, grpcMethod string)
}

type logrusMetricReporter struct {
	metrics *LogsMetrics
}

func (r logrusMetricReporter) Inc(lv logrus.Level) {
	r.metrics.logLevels.With(prometheus.Labels{
		"level": strings.ToLower(lv.String()),
	}).Inc()
}

func (r logrusMetricReporter) IncHTTP(lv logrus.Level, method, handler string) {
	r.metrics.httpLogLevels.With(prometheus.Labels{
		"level":   strings.ToLower(lv.String()),
		"method":  strings.ToLower(method),
		"handler": strings.ToLower(handler),
	}).Inc()
}

func (r logrusMetricReporter) IncRPC(lv logrus.Level, grpcService, grpcMethod string) {
	r.metrics.grpcLogLevels.With(prometheus.Labels{
		"level":        strings.ToLower(lv.String()),
		"grpc_service": strings.ToLower(grpcService),
		"grpc_method":  strings.ToLower(grpcMethod),
	}).Inc()
}

type logrusHook struct {
	reporter logrusReporter
}

func (h logrusHook) Levels() []logrus.Level {
	return defaultLogrusHookLevels
}

func (h logrusHook) Fire(entry *logrus.Entry) error {
	lv := entry.Level
	h.reporter.Inc(lv)

	if service, ok := entry.Data["grpc_service"].(string); ok {
		if method, ok := entry.Data["grpc_method"].(string); ok {
			h.reporter.IncRPC(lv, service, method)
		}
	} else if service, ok = entry.Data["handler"].(string); ok {
		if method, ok := entry.Data["method"].(string); ok {
			h.reporter.IncHTTP(lv, service, method)
		}
	}

	return nil
}
