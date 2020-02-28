package senders

import (
	"fmt"
	"github.com/moira-alert/moira"
	"time"
)

func BuildTitle(events moira.NotificationEvents, trigger moira.TriggerData, frontURI string) string {
	title := fmt.Sprintf("*%s*", events.GetSubjectState())
	triggerURI := trigger.GetTriggerURI(frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" <%s|%s>", triggerURI, trigger.Name)
	} else if trigger.Name != "" {
		title += " " + trigger.Name
	}

	tags := trigger.GetTags()
	if tags != "" {
		title += " " + tags
	}

	title += "\n"
	return title
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if n is negative buildEventsString does not limit the events string
func BuildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool, location *time.Location) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, *fix your system or tune this trigger* to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += "```"
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if msg := event.CreateMessage(location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune("```")) + len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "```"

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}
