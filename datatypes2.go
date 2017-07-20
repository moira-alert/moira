package moira

type TriggerChecksData struct {
	TriggerData
	Ttl             int64        `json:"ttl"`
	TtlState        string       `json:"ttl_state"`
	Throttling      int64        `json:"throttling"`
	IsSimpleTrigger bool         `json:"is_simple_trigger"`
	LastCheck       CheckData    `json:"last_check"`
	Patterns        []string     `json:"patterns"`
	Schedule        ScheduleData `json:"sched"`
	TagsData        []string     `json:"tags"`
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
