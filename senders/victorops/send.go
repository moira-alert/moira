package victorops

import (
	"fmt"
	"strings"
	"time"

	stripmd "github.com/writeas/go-strip-markdown"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/victorops/api"
)

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	createAlertRequest := sender.buildCreateAlertRequest(events, trigger, throttled, plots, time.Now().Unix())
	err := sender.client.CreateAlert(contact.Value, createAlertRequest)
	if err != nil {
		return fmt.Errorf("error while sending alert to victorops: %s", err)
	}
	return nil
}

func (sender *Sender) buildCreateAlertRequest(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool, plots [][]byte, time int64) api.CreateAlertRequest {

	triggerURI := trigger.GetTriggerURI(sender.frontURI)

	createAlertRequest := api.CreateAlertRequest{
		MessageType:       sender.getMessageType(events),
		StateMessage:      sender.buildMessage(events, trigger, throttled),
		EntityDisplayName: sender.buildTitle(events, trigger),
		StateStartTime:    events[len(events)-1].Timestamp,
		TriggerURL:        triggerURI,
		Timestamp:         time,
		MonitoringTool:    "Moira",
		EntityID:          trigger.ID,
	}

	if len(plots) > 0 && sender.imageStoreConfigured {
		imageLink, err := sender.imageStore.StoreImage(plots[0])
		if err != nil {
			sender.logger.Warningf("could not store the plot image in the image store: %s", err)
		} else {
			createAlertRequest.ImageURL = imageLink
		}
	}

	return createAlertRequest
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message strings.Builder
	desc := stripmd.Strip(trigger.Desc)
	eventsString := sender.buildEventsString(events, -1, throttled)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (sender *Sender) getMessageType(events moira.NotificationEvents) api.MessageType {
	msgType := api.Recovery
	for _, event := range events {
		if event.State == moira.StateERROR || event.State == moira.StateEXCEPTION {
			msgType = api.Critical
		}
		if msgType != api.Critical && (event.State == moira.StateWARN || event.State == moira.StateNODATA) {
			msgType = api.Warning
		}
	}
	return msgType
}

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	title := string(events.GetSubjectState())
	title += " " + trigger.Name

	tags := trigger.GetTags()
	if tags != "" {
		title += " " + tags
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
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricsValues(), event.OldState, event.State)
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
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

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}
