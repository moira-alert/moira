package templating

type trigger struct {
	Name string
}

type Event struct {
	Metric         string
	MetricElements []string
	Timestamp      int64
	Value          *float64
	State          string
}

func (event Event) TimestampDecrease(second int64) int64 {
	return event.Timestamp - second
}

func (event Event) TimestampIncrease(second int64) int64 {
	return event.Timestamp + second
}

type triggerDescriptionPopulater struct {
	Trigger *trigger
	Events  []Event
}

func NewTriggerDescriptionPopulater(triggerName string, events []Event) *triggerDescriptionPopulater {
	return &triggerDescriptionPopulater{
		Trigger: &trigger{
			Name: triggerName,
		},
		Events: events,
	}
}

func (templateData *triggerDescriptionPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}
