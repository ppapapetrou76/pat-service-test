package healthcheck

import (
	"context"
	"errors"
)

// Daemon represents a interface that could get the running state of a daemon
type Daemon interface {
	Serving() bool
}

// DaemonServingCheck returns a function that checks the serving state of the given daemon
func DaemonServingCheck(daemon Daemon) CheckingFunc {
	if daemon == nil {
		return func(context.Context) (*CheckingState, error) {
			return nil, errors.New("daemon is nil")
		}
	}

	return func(ctx context.Context) (*CheckingState, error) {
		if daemon.Serving() {
			return &CheckingState{
				State:  StateHealthy,
				Output: "Daemon is serving",
			}, nil
		}
		return &CheckingState{
			State:  StateUnhealthy,
			Output: "Daemon is not serving",
		}, nil
	}
}
