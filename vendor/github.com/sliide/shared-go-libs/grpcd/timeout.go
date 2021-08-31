package grpcd

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Timeout returns a unary interceptor that sets up context deadline for each unary call.
func Timeout(dt time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		l := Logger(ctx)

		if err := ctx.Err(); err != nil {
			l.WithError(err).Warn("Caught canceled before processing the request")

			return nil, status.Error(codes.Canceled, "Canceled by caller")
		}

		if dt <= 0 {
			return handler(ctx, req)
		}

		t := time.Now()
		panicChan := make(chan interface{}, 1)
		respChan := make(chan *unaryResponse, 1)

		ctx2, cancel := context.WithTimeout(ctx, dt)
		defer cancel()

		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicChan <- p
				}
			}()

			resp, err := handler(ctx, req)
			respChan <- &unaryResponse{
				Response: resp,
				Error:    err,
			}
		}()

		select {
		case r := <-panicChan:
			panic(r)
		case <-ctx2.Done():

			// Check if parent content canceled, then marks canceled by caller instead of timeout
			if err := ctx.Err(); err != nil {
				l.WithError(err).WithFields(logrus.Fields{
					"timeout":  dt.Seconds(),
					"duration": time.Since(t).Seconds(),
				}).Warn("Caught canceled while processing the request")

				return nil, status.Error(codes.Canceled, "Canceled by caller")
			}

			// Timeout error
			l.WithError(ctx2.Err()).WithFields(logrus.Fields{
				"timeout":  dt.Seconds(),
				"duration": time.Since(t).Seconds(),
			}).Warn("Caught timeout while processing the request")

			return nil, status.Error(codes.DeadlineExceeded, "Deadline exceeded")
		case r := <-respChan:
			return r.Response, r.Error
		}
	}
}

type unaryResponse struct {
	Response interface{}
	Error    error
}
