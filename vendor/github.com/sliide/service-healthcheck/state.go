package healthcheck

import (
	"encoding/json"
)

// State represents the health check state
type State int

var stateToName = map[int]string{
	0: "healthy",
	1: "degraded",
	2: "unhealthy",
	3: "unknown",
}

var nameToState = map[string]State{
	"healthy":   StateHealthy,
	"degraded":  StateDegraded,
	"unhealthy": StateUnhealthy,
	"unknown":   StateUnknown,
}

// MarshalJSON returns the JSON encoding of the state.
func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON returns the JSON encoding of the state.
func (s *State) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	if v, ok := nameToState[str]; ok {
		*s = v
		return nil
	}
	*s = StateUnknown
	return nil
}

func (s State) String() string {
	return stateToName[int(s)]
}

const (

	// StateHealthy represents in the healthy state
	StateHealthy State = 0

	// StateDegraded represents in the degraded state, it's ok but not very well
	StateDegraded State = 1

	// StateUnhealthy represents in the unhealthy state
	StateUnhealthy State = 2

	// StateUnknown represents in the unknown state
	StateUnknown State = 3
)
