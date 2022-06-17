package clock

import "time"

// SystemClock is struct clock-component.
type SystemClock struct{}

// NewSystemClock is construct for clock-component.
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// Now returns time.Time.
func (t *SystemClock) Now() time.Time {
	return time.Now().UTC()
}

// Sleep pauses the current goroutine for at least the passed duration
func (t *SystemClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}
