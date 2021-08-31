package grpcd

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ctxLoggerKey struct{}

// Logger returns a logrus entry from the context
// Always returns a logger.
func Logger(ctx context.Context) *logrus.Entry {
	logger, ok := ctx.Value(ctxLoggerKey{}).(*logrus.Entry)
	if !ok {
		return logrus.NewEntry(logrus.StandardLogger())
	}

	return logger
}

// NewContextWithLogger returns a new context which sets the logger passed in.
func NewContextWithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, ctxLoggerKey{}, logger)
}

// Logging setup logger for all unary request
//
// The reason splitting logger and entry into tow interceptor functions is for the testing,
// we want to output to a buffer when testing interceptor.
func Logging(logger *logrus.Entry) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = NewContextWithLogger(ctx, logger)

		return handler(ctx, req)
	}
}
