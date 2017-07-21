package moira

type Trigger struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Desc            string       `json:"desc"`
	Targets         []string     `json:"targets"`
	WarnValue       float64      `json:"warn_value"`
	ErrorValue      float64      `json:"error_value"`
	Tags            []string     `json:"tags"`
	TtlState        string       `json:"ttl_state"`
	Ttl             int64        `json:"ttl"`
	Schedule        ScheduleData `json:"sched"`
	Expression      string       `json:"expression"`
	Patterns        []string     `json:"patterns"`
	IsSimpleTrigger bool         `json:"is_simple_trigger"`
}

type TriggerChecks struct {
	Trigger
	Throttling int64     `json:"throttling"`
	LastCheck  CheckData `json:"last_check"`
}

type CheckData struct {
	Metrics   map[string]MetricData `json:"metrics"`
	Score     int64                 `json:"score"`
	State     string                `json:"state"`
	Timestamp int64                 `json:"timestamp"`
}

type MetricData struct {
	EventTimestamp int64  `json:"event_timestamp"`
	State          string `json:"state"`
	Suppressed     bool   `json:"suppressed"`
	Timestamp      int64  `json:"timestamp"`
}
