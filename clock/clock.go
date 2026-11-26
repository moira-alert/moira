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

// ScheduleLocation returns a fixed timezone for given schedule offset. converts tzOffset in minutes and the converts the sign, cause in schedule it is stored with "-" for eastern regions and "+" for western.
func ScheduleLocation(tzOffset int64) *time.Location {
	return time.FixedZone("", int(-tzOffset*60))
}

// NextLocalMidnight returns the start of the next local day relative to timestamp in the zone difined by tzOffset.
func NextLocalMidnight(timestamp time.Time, tzOffset int64) time.Time {
	location := ScheduleLocation(tzOffset)
	local := timestamp.In(location)
	year, month, day := local.Date()

	return time.Date(year, month, day+1, 0, 0, 0, 0, location)
}
