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

// NowUnix returns current time in a Unix time format.
func (t *SystemClock) NowUnix() int64 {
	return time.Now().Unix()
}
