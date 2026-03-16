package proxy

import "time"

// Event is a timestamped process event emitted by the manager runtime.
type Event struct {
	Time    time.Time `json:"time"`
	Process string    `json:"process"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
}

// ProcessStatus is a snapshot of runtime state and counters for one process.
type ProcessStatus struct {
	Name          string    `json:"name"`
	Port          int       `json:"port"`
	Command       string    `json:"command"`
	Running       bool      `json:"running"`
	PID           int       `json:"pid"`
	Restarts      int       `json:"restarts"`
	LastError     string    `json:"lastError,omitempty"`
	StartedAt     time.Time `json:"startedAt"`
	StoppedAt     time.Time `json:"stoppedAt"`
	Requests      int64     `json:"requests"`
	Responses     int64     `json:"responses"`
	Notifications int64     `json:"notifications"`
}
