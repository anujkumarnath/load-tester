package requester

import "time"

type Result struct {
	Duration   time.Duration
	StatusCode int
	Bytes      int
	Error      string
	Timestamp  time.Time
}
