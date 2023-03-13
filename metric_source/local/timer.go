package local

// Timer is responsible for managing time ranges and metrics' timeslots
type Timer struct {
	startTime int64
	stopTime  int64
	stepTime  int64
}

func RoundTimestamps(from, until, retention int64) (roundedFrom, roundedUntil int64) {
	return CeilToMultiplier(from, retention), FloorToMultiplier(until, retention) + retention
}

func NewTimerRoundingTimestamps(startTime int64, stopTime int64, retention int64) Timer {
	startTime, stopTime = RoundTimestamps(startTime, stopTime, retention)
	return Timer{
		startTime: startTime,
		stopTime:  stopTime,
		stepTime:  retention,
	}
}

func (t Timer) NumberOfTimeSlots() int {
	return t.GetTimeSlot(t.stopTime)
}

func (t Timer) GetTimeSlot(timestamp int64) int {
	timeSlot := FloorToMultiplier(timestamp-t.startTime, t.stepTime) / t.stepTime
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
