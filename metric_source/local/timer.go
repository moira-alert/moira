package local

type Timer struct {
	from      int64
	until     int64
	retention int64
}

func NewTimerRoundingTimestamps(from int64, until int64, retention int64) *Timer {
	return &Timer{
		from:      CeilToMultiplier(from, retention),
		until:     FloorToMultiplier(until, retention) + retention,
		retention: retention,
	}
}

func (t Timer) NumberOfTimeSlots() int {
	return t.GetTimeSlot(t.until)
}

func (t Timer) GetTimeSlot(timestamp int64) int {
	retentionFrom := CeilToMultiplier(t.from, t.retention)
	timeSlot := int((timestamp - retentionFrom) / t.retention)
	return timeSlot
}

func CeilToMultiplier(ts, retention int64) int64 {
	if (ts % retention) == 0 {
		return ts
	}
	return (ts + retention) / retention * retention
}

func FloorToMultiplier(ts, retention int64) int64 {
	return ts / retention * retention
}
