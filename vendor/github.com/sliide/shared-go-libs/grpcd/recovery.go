package grpcd

import (
	"context"
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Recovery returns a new unary server interceptor for panic recovery.
func Recovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		defer func() {
			if p := recover(); p != nil {
				stack := fmt.Sprintf("%s", debug.Stack())
				logger := Logger(ctx)

				logger.WithField("stacktrace", stack).
					WithError(fmt.Errorf("%v", p)).
					Error("Caught panic in request")

				err = status.Errorf(codes.Internal, "InternalServerError")
			}
		}()

		return handler(ctx, req)
	}
}
