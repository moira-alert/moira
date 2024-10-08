package local

type timer struct {
	startTime int64
	stopTime  int64
	stepTime  int64
}

func roundTimestamps(startTime, stopTime, retention int64) (roundedStart, roundedStop int64) {
	until := floorToMultiplier(stopTime, retention) + retention
	from := ceilToMultiplier(startTime, retention)

	return from, until
}

func newTimerRoundingTimestamps(startTime int64, stopTime int64, retention int64) timer {
	startTime, stopTime = roundTimestamps(startTime, stopTime, retention)
	return timer{
		startTime: startTime,
		stopTime:  stopTime,
		stepTime:  retention,
	}
}

func (t timer) numberOfTimeSlots() int {
	return t.getTimeSlot(t.stopTime)
}

func (t timer) getTimeSlot(timestamp int64) int {
	timeSlot := floorToMultiplier(timestamp-t.startTime, t.stepTime) / t.stepTime
	return int(timeSlot)
}

func ceilToMultiplier(ts, retention int64) int64 {
	if (ts % retention) == 0 {
		return ts
	}
	return (ts + retention) / retention * retention
}

func floorToMultiplier(ts, retention int64) int64 {
	if ts < 0 {
		ts -= retention - 1
	}
	return ts - ts%retention
}
