package healthcheck

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const (
	// DefaultTimeout represents a default timeout of checking
	DefaultTimeout = time.Second * 10
)

// CheckingState represents the health checking result which returns from CheckingFunc
type CheckingState struct {
	State  State
	Output string // Extra output for further debug
}

// CheckingResult represents the health checking detail which be used in CheckingResults
type CheckingResult struct {
	State    State         `json:"state"`
	Output   string        `json:"output"`
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
}

// CheckingResults represents the health checking result which returns from HealthChecker
type CheckingResults struct {
	Checks   []CheckingResult `json:"checks"`
	Duration time.Duration    `json:"duration"`
	Runtime  Runtime          `json:"runtime"`
}

// Runtime represents runtime information of the service
type Runtime struct {
	Host        string `json:"host"`
	GoVersion   string `json:"go_version"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// IsHealthy returns the is in healthy state
func (c CheckingResults) IsHealthy() bool {
	checks := c.Checks
	if len(checks) <= 0 {
		return false
	}
	for _, check := range checks {
		if check.State != StateHealthy && check.State != StateDegraded {
			return false
		}
	}
	return true
}

// IsDegraded returns the is degraded state
func (c CheckingResults) IsDegraded() bool {
	checks := c.Checks
	if len(checks) <= 0 {
		return false
	}
	for _, check := range checks {
		if check.State == StateDegraded {
			return true
		}
	}
	return false
}

// GetState provides the total states based on an individual items of the results.
// Be aware that it's not compatible with IsHealthy() and IsDegraded().
// If no checks are provided, it's considered unhealthy.
func (c CheckingResults) GetState() State {
	checks := c.Checks
	if len(checks) <= 0 {
		return StateUnhealthy
	}

	healthy := true
	degraded := true

	for _, check := range checks {
		healthy = healthy && check.State == StateHealthy
		degraded = degraded && (check.State == StateHealthy || check.State == StateDegraded)
	}

	if healthy {
		return StateHealthy
	}

	if degraded {
		return StateDegraded
	}

	return StateUnhealthy
}

// CheckingFunc defines a health-checking function
type CheckingFunc func(context.Context) (*CheckingState, error)

// HealthChecker defines an interface of health checker
type HealthChecker interface {
	AddCheck(name string, f CheckingFunc)
	RunChecks(context.Context) CheckingResults
}

// Params represents parameters of checker
type Params struct {
	Timeout     time.Duration
	Service     string
	Environment string
	Version     string
}

// New returns a new HealthChecker
func New(param Params) HealthChecker {
	hostname, _ := os.Hostname()
	return &healthcheck{
		Timeout:     param.Timeout,
		Service:     param.Service,
		Environment: param.Environment,
		GoVersion:   runtime.Version(),
		Version:     param.Version,
		Host:        hostname,
	}
}

type healthcheck struct {
	Timeout     time.Duration
	Service     string
	Environment string
	Version     string
	Host        string
	GoVersion   string

	Checks []checkFunc
}

type checkFunc struct {
	Name string
	Func CheckingFunc
}

func (h *healthcheck) AddCheck(name string, f CheckingFunc) {
	h.Checks = append(h.Checks, checkFunc{
		Name: name,
		Func: f,
	})
}

func (h healthcheck) RunChecks(ctx context.Context) CheckingResults {
	if len(h.Checks) <= 0 {
		return CheckingResults{
			Runtime: h.runtime(),
		}
	}

	timeout := h.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	t := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ress := make([]CheckingResult, len(h.Checks))
	wg := sync.WaitGroup{}
	for idx, check := range h.Checks {
		wg.Add(1)
		go func(idx int, check checkFunc) {
			defer wg.Done()

			t := time.Now()
			r := runCheck(ctx, timeout, check.Func)

			ress[idx] = CheckingResult{
				Name:     check.Name,
				Duration: time.Since(t),
				State:    r.State,
				Output:   r.Output,
			}

		}(idx, check)
	}

	wg.Wait()
	return CheckingResults{
		Checks:   ress,
		Duration: time.Since(t),
		Runtime:  h.runtime(),
	}
}

func (h healthcheck) runtime() Runtime {
	return Runtime{
		Host:        h.Host,
		GoVersion:   h.GoVersion,
		Service:     h.Service,
		Environment: h.Environment,
		Version:     h.Version,
	}
}

type checkingState struct {
	State  State
	Output string
	Error  error
}

func runCheck(ctx context.Context, timeout time.Duration, f CheckingFunc) checkingState {
	doneChan := make(chan checkingState)
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go func() {
		defer func() {
			if p := recover(); p != nil {
				doneChan <- checkingState{
					State:  StateUnhealthy,
					Output: fmt.Sprintf("checking panicked: %v, stacktrace: %s", p, debug.Stack()),
					Error:  fmt.Errorf("checking panicked: %v", p),
				}
			}
		}()

		r, err := f(ctxTimeout)
		if err != nil {
			doneChan <- checkingState{
				State:  StateUnhealthy,
				Output: fmt.Sprintf("checking failed: %v", err),
				Error:  fmt.Errorf("checking failed: %w", err),
			}
			return
		}
		doneChan <- checkingState{
			State:  r.State,
			Output: r.Output,
		}
	}()

	select {
	case <-ctxTimeout.Done():
		err := ctxTimeout.Err()
		if err != ctx.Err() {
			return checkingState{
				State:  StateUnknown,
				Output: fmt.Sprintf("checking timeout: %v", ctxTimeout.Err()),
				Error:  fmt.Errorf("checking timeout: %w", ctxTimeout.Err()),
			}
		}
		return checkingState{
			State:  StateUnknown,
			Output: fmt.Sprintf("checking canceled by caller: %v", err),
			Error:  fmt.Errorf("checking canceled by caller:%w", ctx.Err()),
		}
	case r := <-doneChan:
		return r
	}
}
