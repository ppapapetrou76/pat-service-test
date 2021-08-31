package prometheus

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	requestInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_unary_requests_in_flight",
			Help: "The current number of gRPC unary requests is being served.",
		},
		[]string{"grpc_service", "grpc_method"},
	)

	requestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_unary_requests_total",
			Help: "Total number of gRPC unary requests made and responded.",
		},
		[]string{"grpc_service", "grpc_method", "grpc_code"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_unary_requests_duration_seconds",
			Help:    "The gRPC unary request latencies in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"grpc_service", "grpc_method"},
	)
)

// NewRPCMetrics returns a metrics which used for monitoring gRPC server.
func NewRPCMetrics() *RPCMetrics {
	return &RPCMetrics{
		inFlight:     requestInFlight,
		total:        requestTotal,
		durationSecs: requestDuration,
	}
}

// RPCMetrics represents a collection of metrics to be registered on a Prometheus metrics registry.
type RPCMetrics struct {
	inFlight *prometheus.GaugeVec
	total    *prometheus.CounterVec

	durationSecs *prometheus.HistogramVec
}

// UnaryServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Unary RPCs.
func (m *RPCMetrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		reporter := newRPCReporter(m, info.FullMethod)
		reporter.IncInFlight()
		defer reporter.DecInFlight()

		resp, err := handler(ctx, req)
		status, _ := status.FromError(err)

		reporter.Handled(status.Code())

		return resp, err
	}
}

func newRPCReporter(m *RPCMetrics, grpcFullMethodName string) *rpcReporter {
	serviceName, methodName := grpcSplitMethodName(grpcFullMethodName)

	return &rpcReporter{
		metrics:     m,
		serviceName: serviceName,
		methodName:  methodName,
		startTime:   time.Now(),
	}
}

type rpcReporter struct {
	metrics     *RPCMetrics
	serviceName string
	methodName  string
	startTime   time.Time
}

func (r rpcReporter) IncInFlight() {
	r.metrics.inFlight.With(r.label()).Inc()
}

func (r rpcReporter) DecInFlight() {
	r.metrics.inFlight.With(r.label()).Dec()
}

func (r rpcReporter) Handled(code codes.Code) {
	r.metrics.total.With(r.labelCode(code)).Inc()
	r.metrics.durationSecs.With(r.label()).Observe(r.since().Seconds())
}

func (r rpcReporter) label() prometheus.Labels {
	return prometheus.Labels{
		"grpc_service": strings.ToLower(r.serviceName),
		"grpc_method":  strings.ToLower(r.methodName),
	}
}

func (r rpcReporter) labelCode(code codes.Code) prometheus.Labels {
	labels := r.label()
	labels["grpc_code"] = strings.ToLower(code.String())

	return labels
}

func (r rpcReporter) since() time.Duration {
	return time.Since(r.startTime)
}

func grpcSplitMethodName(fullMethodName string) (grpcService, grpcMethod string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}

	return "unknown", fullMethodName
}
