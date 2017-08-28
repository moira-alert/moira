package moira

import "math"

type Trigger struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Desc            *string       `json:"desc,omitempty"`
	Targets         []string      `json:"targets"`
	WarnValue       *float64      `json:"warn_value"`
	ErrorValue      *float64      `json:"error_value"`
	Tags            []string      `json:"tags"`
	TtlState        *string       `json:"ttl_state,omitempty"`
	Ttl             *int64        `json:"ttl"`
	Schedule        *ScheduleData `json:"sched,omitempty"`
	Expression      *string       `json:"expression,omitempty"`
	Patterns        []string      `json:"patterns"`
	IsSimpleTrigger bool          `json:"is_simple_trigger"`
}

type TriggerChecks struct {
	Trigger
	Throttling int64     `json:"throttling"`
	LastCheck  CheckData `json:"last_check"`
}

type CheckData struct {
	Metrics        map[string]MetricState `json:"metrics"`
	Score          int64                  `json:"score"`
	State          string                 `json:"state"`
	Timestamp      int64                  `json:"timestamp,omitempty"`
	EventTimestamp int64                  `json:"event_timestamp,omitempty"`
	Suppressed     bool                   `json:"suppressed,omitempty"`
	Message        string                 `json:"msg,omitempty"`
}

type MetricState struct {
	EventTimestamp int64    `json:"event_timestamp"`
	State          string   `json:"state"`
	Suppressed     bool     `json:"suppressed"`
	Timestamp      int64    `json:"timestamp"`
	Value          *float64 `json:"value,omitempty"`
	Maintenance    int64    `json:"maintenance,omitempty"`
}

type MetricEvent struct {
	Metric  string `json:"metric"`
	Pattern string `json:"pattern"`
}

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

func (metricState *MetricState) GetCheckPoint(checkPointGap int64) int64 {
	return int64(math.Max(float64(metricState.Timestamp-checkPointGap), float64(metricState.EventTimestamp)))
}

func (metricState MetricState) GetEventTimestamp() int64 {
	if metricState.EventTimestamp == 0 {
		return metricState.Timestamp
	}
	return metricState.EventTimestamp
}

func (checkData CheckData) GetEventTimestamp() int64 {
	if checkData.EventTimestamp == 0 {
		return checkData.Timestamp
	}
	return checkData.EventTimestamp
}
