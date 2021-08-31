package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Pinger represents a interface that could ping a service to check the connection and status
type Pinger interface {
	Ping(context.Context) error
}

// PingFunc represents a ping function that implements Pinger interface
type PingFunc func(context.Context) error

func (f PingFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

const (
	defaultAcceptablePing = time.Millisecond * 100
)

// PingCheck returns a checking function that checks the connection state of the given sql database connection
func PingCheck(pinger Pinger, acceptablePing ...time.Duration) CheckingFunc {
	if pinger == nil {
		return func(context.Context) (*CheckingState, error) {
			return nil, errors.New("pinger is nil")
		}
	}

	limit := defaultAcceptablePing
	if len(acceptablePing) >= 1 {
		limit = acceptablePing[0]
	}

	return func(ctx context.Context) (*CheckingState, error) {
		t := time.Now()
		err := pinger.Ping(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to ping: %w", err)
		} else if duration := time.Since(t); duration > limit {
			return &CheckingState{
				State:  StateDegraded,
				Output: fmt.Sprintf("OK, but response time was over %.2f seconds", duration.Seconds()),
			}, nil
		}
		return &CheckingState{
			State:  StateHealthy,
			Output: "OK",
		}, nil
	}
}
