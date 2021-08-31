package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/status"
)

// GRPCConnectionCheck returns a function that checks the connection state by calling gRPC client's Echo function
func GRPCConnectionCheck(client echo.EchoClient, acceptablePing ...time.Duration) CheckingFunc {
	if client == nil {
		return func(context.Context) (*CheckingState, error) {
			return nil, errors.New("client is nil")
		}
	}

	limit := defaultAcceptablePing
	if len(acceptablePing) >= 1 {
		limit = acceptablePing[0]
	}

	return func(ctx context.Context) (*CheckingState, error) {
		t := time.Now()
		_, err := client.UnaryEcho(ctx, &echo.EchoRequest{
			Message: "Knock! knock! do you want to build a snowman?",
		})
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, fmt.Errorf("unexpected gRPC error response: %w", err)
			}
			if s.Code() == codes.Unavailable {
				return &CheckingState{
					State:  StateUnhealthy,
					Output: "Service is unavailable",
				}, nil
			}
			return &CheckingState{
				State:  StateUnhealthy,
				Output: fmt.Sprintf("Service is connected, but non-ok response: %v", err),
			}, nil
		} else if since := time.Since(t); since >= limit {
			return &CheckingState{
				State:  StateDegraded,
				Output: fmt.Sprintf("OK, but response time was over %.2f seconds", since.Seconds()),
			}, nil
		}
		return &CheckingState{
			State:  StateHealthy,
			Output: "OK",
		}, nil
	}
}
