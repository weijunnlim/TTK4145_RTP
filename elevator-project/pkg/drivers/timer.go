// Translating `timer.c` into Go as `pkg/drivers/timer.go`

package drivers

import (
	"time"
)

var (
	timerEndTime time.Time
	timerActive  bool
)

// Start initializes the timer with a specified duration in seconds.
func Start(duration float64) {
	timerEndTime = time.Now().Add(time.Duration(duration * float64(time.Second)))
	timerActive = true
}

// Stop deactivates the timer.
func Stop() {
	timerActive = false
}

// TimedOut checks whether the timer has expired.
func TimedOut() bool {
	return timerActive && time.Now().After(timerEndTime)
}
