package local

// Timer is responsible for managing time ranges and metrics' timeslots
type Timer struct {
	startTime int64
	stopTime  int64
	stepTime  int64
}

// Rounds start and stop time in a specific manner requered by carbonapi
func RoundTimestamps(startTime, stopTime, retention int64) (roundedStart, roundedStop int64) {
	return ceilToMultiplier(startTime, retention), floorToMultiplier(stopTime, retention) + retention
}

// Creates new timer rounding start and stop time in a specific manner requered by carbonapi
// Timers should be created only with this function
func NewTimerRoundingTimestamps(startTime int64, stopTime int64, retention int64) Timer {
	startTime, stopTime = RoundTimestamps(startTime, stopTime, retention)
	return Timer{
		startTime: startTime,
		stopTime:  stopTime,
		stepTime:  retention,
	}
}

// Returns the number of timeslots from this timer's startTime until its stopTime with it's retention
func (t Timer) NumberOfTimeSlots() int {
	return t.GetTimeSlot(t.stopTime)
}

// Returns the index of given timestamp (rounded by timestamp) in this timer's time range
func (t Timer) GetTimeSlot(timestamp int64) int {
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
