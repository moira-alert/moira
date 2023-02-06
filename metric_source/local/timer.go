package local

type Timer struct {
	from                  int64
	until                 int64
	retention             int64
	allowRealTimeAlerting bool
}

func MakeTimer(from int64, until int64, retention int64, allowRealTimeAlerting bool) Timer {
	return Timer{
		from:                  from,
		until:                 until,
		retention:             retention,
		allowRealTimeAlerting: allowRealTimeAlerting,
	}
}

func (t Timer) NumberOfTimeSlots() int {
	return t.GetTimeSlot(t.until)
}

func (t Timer) GetTimeSlot(timestamp int64) int {
	retentionFrom := divideRoundingToCeiling(t.from, t.retention)
	timeSlot := int((timestamp - retentionFrom) / t.retention)
	return timeSlot
}

func divideRoundingToCeiling(ts, retention int64) int64 {
	if (ts % retention) == 0 {
		return ts
	}
	return (ts + retention) / retention * retention
}
