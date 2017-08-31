package moira

import "math"

//Trigger represents trigger data object
type Trigger struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Desc            *string       `json:"desc,omitempty"`
	Targets         []string      `json:"targets"`
	WarnValue       *float64      `json:"warn_value"`
	ErrorValue      *float64      `json:"error_value"`
	Tags            []string      `json:"tags"`
	TTLState        *string       `json:"ttl_state,omitempty"`
	TTL             *int64        `json:"ttl"`
	Schedule        *ScheduleData `json:"sched,omitempty"`
	Expression      *string       `json:"expression,omitempty"`
	Patterns        []string      `json:"patterns"`
	IsSimpleTrigger bool          `json:"is_simple_trigger"`
}

//TriggerCheck represent trigger data with last check data and check timestamp
type TriggerCheck struct {
	Trigger
	Throttling int64     `json:"throttling"`
	LastCheck  CheckData `json:"last_check"`
}

//CheckData represent last trigger check data
type CheckData struct {
	Metrics        map[string]MetricState `json:"metrics"`
	Score          int64                  `json:"score"`
	State          string                 `json:"state"`
	Timestamp      int64                  `json:"timestamp,omitempty"`
	EventTimestamp int64                  `json:"event_timestamp,omitempty"`
	Suppressed     bool                   `json:"suppressed,omitempty"`
	Message        string                 `json:"msg,omitempty"`
}

//MetricState represent metric state data for given timestamp
type MetricState struct {
	EventTimestamp int64    `json:"event_timestamp"`
	State          string   `json:"state"`
	Suppressed     bool     `json:"suppressed"`
	Timestamp      int64    `json:"timestamp"`
	Value          *float64 `json:"value,omitempty"`
	Maintenance    int64    `json:"maintenance,omitempty"`
}

//MetricEvent represent cache metric new event
type MetricEvent struct {
	Metric  string `json:"metric"`
	Pattern string `json:"pattern"`
}

//GetOrCreateMetricState gets metric state from check data or create new if CheckData has no state for given metric
func (checkData *CheckData) GetOrCreateMetricState(metric string, emptyTimestampValue int64) MetricState {
	_, ok := checkData.Metrics[metric]
	if !ok {
		checkData.Metrics[metric] = MetricState{
			State:     "NODATA",
			Timestamp: emptyTimestampValue,
		}
	}
	return checkData.Metrics[metric]
}

//GetCheckPoint gets check point for given MetricState
//CheckPoint is the timestamp from which to start checking the current state of the metric
func (metricState *MetricState) GetCheckPoint(checkPointGap int64) int64 {
	return int64(math.Max(float64(metricState.Timestamp-checkPointGap), float64(metricState.EventTimestamp)))
}

//GetEventTimestamp gets event timestamp for given metric
func (metricState MetricState) GetEventTimestamp() int64 {
	if metricState.EventTimestamp == 0 {
		return metricState.Timestamp
	}
	return metricState.EventTimestamp
}

//GetEventTimestamp gets event timestamp for given check
func (checkData CheckData) GetEventTimestamp() int64 {
	if checkData.EventTimestamp == 0 {
		return checkData.Timestamp
	}
	return checkData.EventTimestamp
}
