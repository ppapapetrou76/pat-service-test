package grpcd

import (
	"google.golang.org/grpc"

	"github.com/sliide/shared-go-libs/metric/prometheus"
)

// Prometheus returns a interceptor that setup prometheus metrics.
func Prometheus() grpc.UnaryServerInterceptor {
	return prometheus.NewRPCMetrics().UnaryServerInterceptor()
}
