package moira

var (
	eventStateWeight = map[string]int{
		"OK":     0,
		"WARN":   1,
		"ERROR":  100,
		"NODATA": 10000,
	}
	eventHighDegradationTag = "HIGH DEGRADATION"
	eventDegradationTag     = "DEGRADATION"
	eventProgressTag        = "PROGRESS"
)

// GetEventTags returns additional subscription tags based on trigger state
func (eventData *NotificationEvent) GetEventTags() []string {
	tags := []string{eventData.State, eventData.OldState}
	if oldStateWeight, ok := eventStateWeight[eventData.OldState]; ok {
		if newStateWeight, ok := eventStateWeight[eventData.State]; ok {
			if newStateWeight > oldStateWeight {
				if newStateWeight-oldStateWeight >= 100 {
					tags = append(tags, eventHighDegradationTag)
				}
				tags = append(tags, eventDegradationTag)
			}
			if newStateWeight < oldStateWeight {
				tags = append(tags, eventProgressTag)
			}
		}
	}
	return tags
}
