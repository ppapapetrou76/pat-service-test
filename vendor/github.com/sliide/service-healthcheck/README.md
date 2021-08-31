# Service HealthCheck

A healthcheck library for go services.

## Usage

See following example:

```go
import (
  "context"
  "net/http"

  "https://github.com/sliide/service-healthcheck"
)

hc := healthcheck.New(healthcheck.CheckerParam{
  Service:     "test-service",
  Environment: "test-environment",
  Version:     "test-version",
})
hc.AddCheck("system-1", func(context.Context) (*CheckingState, error) {
  return &CheckingState{
    State: Healthy,
    Output: "all good",
  }, nil
})
hc.AddCheck("system-2", func() (*CheckResult, error) {
  return &CheckResult{
    State: Unhealthy,
    Output: "failed to connect to the instance",
  }, nil
})

h := healthcheck.Handler(hc)
http.HandleFunc("/healthcheck", h)
http.ListenAndServe(":8080", nil)
```

## States

The following strings are the possible values in the output checks,
please see the example output.

- `healthy`: OK
- `degraded`: Warning
- `unhealthy`: Critical
- `unknown`: Unknown

## Example Output

```json
{
  "checks":[
    {
        "state":"healthy",
        "output":"ok",
        "name":"check-ok",
        "duration":19781
    },
    {
        "state":"degraded",
        "output":"not-well",
        "name":"check-warning",
        "duration":12234002110
    },
    {
        "state":"unhealthy",
        "output":"dead",
        "name":"check-critical",
        "duration":5100211264
    },
    {
        "state":"unknown",
        "output":"what's happening?",
        "name":"check-unknown",
        "duration":190422404
    }
  ],
  "duration":13000031834,
  "duration_in_seconds":13.000031834,
  "runtime": {
    "host": "siyuan-mac",
    "go_version": "go1.13.7",
    "environment": "dev",
    "version": "dev-07-02-2020"
  },
  "is_healthy":true,
  "is_degraded":false
}
```

## Checking

```go
import (
  "context"
  "log"

  "https://github.com/sliide/service-healthcheck"
)

c := healthcheck.Client{
  Target: "https://example.com/healthcheck",
}
result, _ := c.Check(context.Background())
healthy := result.IsHealthy()
degraded := result.IsDegraded()
```
