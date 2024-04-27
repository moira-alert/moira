package clock

import "time"

// SystemClock is struct clock-component.
type SystemClock struct{}

// NewSystemClock is construct for clock-component.
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// Now returns now time.Time with UTC location.
func (t *SystemClock) NowUTC() time.Time {
	return time.Now().UTC()
}

// Now returns now time.Time as a Unix time.
func (t *SystemClock) NowUnix() int64 {
	return time.Now().Unix()
}
