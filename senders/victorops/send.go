package victorops

import (
	"fmt"
	"strings"

	"github.com/writeas/go-strip-markdown"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/victorops/api"
)

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	var messageType api.MessageType
	switch events[len(events)-1].State {
	case moira.StateERROR:
		messageType = api.Critical
	case moira.StateEXCEPTION:
		messageType = api.Critical
	case moira.StateWARN:
		messageType = api.Warning
	case moira.StateOK:
		messageType = api.Recovery
	case moira.StateNODATA:
		messageType = api.Info

	}

	createAlertRequest := api.CreateAlertRequest{
		MessageType:       messageType,
		StateMessage:      sender.buildMessage(events, trigger, throttled),
		EntityDisplayName: sender.buildTitle(events, trigger),
		StateStartTime:    events[len(events)-1].Timestamp,
	}
	err := sender.client.CreateAlert(contact.Value, createAlertRequest)
	if err != nil {
		return fmt.Errorf("error while sending alert to victorops: %s", err)
	}
	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message strings.Builder
	desc := stripmd.Strip(trigger.Desc)
	eventsString := sender.buildEventsString(events, -1, throttled)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	title := fmt.Sprintf("%s", string(events.GetSubjectState()))

	tags := trigger.GetTags()
	if tags != "" {
		title += " " + tags
	}

	triggerURI := trigger.GetTriggerURI(sender.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" %s|%s", triggerURI, trigger.Name)
	} else if trigger.Name != "" {
		title += " " + trigger.Name
	}
	title += "\n"

	return title
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if n is negative buildEventsString does not limit the events string
func (sender *Sender) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, fix your system or tune this trigger to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
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

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}
