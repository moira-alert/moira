package local

type Timer struct {
	from      int64
	until     int64
	retention int64
}

func RoundTimestamps(from, until, retention int64) (roundedFrom, roundedUntil int64) {
	return CeilToMultiplier(from, retention), FloorToMultiplier(until, retention) + retention
}

func NewTimerRoundingTimestamps(from int64, until int64, retention int64) *Timer {
	from, until = RoundTimestamps(from, until, retention)
	return &Timer{
		from:      from,
		until:     until,
		retention: retention,
	}
}

func (t Timer) NumberOfTimeSlots() int {
	return t.GetTimeSlot(t.until)
}

func (t Timer) GetTimeSlot(timestamp int64) int {
	timeSlot := FloorToMultiplier(timestamp-t.from, t.retention) / t.retention
	return int(timeSlot)
}

func CeilToMultiplier(ts, retention int64) int64 {
	if (ts % retention) == 0 {
		return ts
	}
	return (ts + retention) / retention * retention
}

func FloorToMultiplier(ts, retention int64) int64 {
	if ts < 0 {
		ts -= retention - 1
	}
	return ts - ts%retention
}
