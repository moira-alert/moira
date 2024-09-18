package clock

import "time"

// SystemClock is struct clock-component.
type SystemClock struct{}

// NewSystemClock is construct for clock-component.
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// NowUTC returns now time.Time with UTC location.
func (t *SystemClock) NowUTC() time.Time {
	return time.Now().UTC()
}

// Sleep pauses the current goroutine for at least the passed duration.
func (t *SystemClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// NowUnix returns now time.Time as a Unix time.
func (t *SystemClock) NowUnix() int64 {
	return time.Now().Unix()
}
