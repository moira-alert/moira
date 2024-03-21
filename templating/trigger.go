package templating

type trigger struct {
	Name string
}

// Event represents a template event with fields allowed for use in templates.
type Event struct {
	Metric         string
	MetricElements []string
	Timestamp      int64
	Value          *float64
	State          string
}

// TimestampDecrease decreases the timestamp of the event by the given number of seconds.
func (event Event) TimestampDecrease(second int64) int64 {
	return event.Timestamp - second
}

// TimestampIncrease increases the timestamp of the event by the given number of seconds.
func (event Event) TimestampIncrease(second int64) int64 {
	return event.Timestamp + second
}

type triggerDescriptionPopulater struct {
	Trigger *trigger
	Events  []Event
}

// NewTriggerDescriptionPopulater creates a new trigger description populater with the given trigger name and template events.
func NewTriggerDescriptionPopulater(triggerName string, events []Event) *triggerDescriptionPopulater {
	return &triggerDescriptionPopulater{
		Trigger: &trigger{
			Name: triggerName,
		},
		Events: events,
	}
}

// Populate populates the given template with trigger description data.
func (templateData *triggerDescriptionPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}
