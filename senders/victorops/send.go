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
	victoropsClient, ok := sender.clients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	createAlertRequest := victoropsClient.buildCreateAlertRequest(events, trigger, throttled, plots, time.Now().Unix())
	err := victoropsClient.client.CreateAlert(contact.Value, createAlertRequest)
	if err != nil {
		return fmt.Errorf("error while sending alert to victorops: %s", err)
	}

	return nil
}

func (client *victoropsClient) buildCreateAlertRequest(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool, plots [][]byte, time int64) api.CreateAlertRequest {
	triggerURI := trigger.GetTriggerURI(client.frontURI)

	createAlertRequest := api.CreateAlertRequest{
		MessageType:       client.getMessageType(events),
		StateMessage:      client.buildMessage(events, trigger, throttled),
		EntityDisplayName: client.buildTitle(events, trigger, throttled),
		StateStartTime:    events[len(events)-1].Timestamp,
		TriggerURL:        triggerURI,
		Timestamp:         time,
		MonitoringTool:    "Moira",
		EntityID:          trigger.ID,
	}

	if len(plots) > 0 && client.imageStoreConfigured {
		imageLink, err := client.imageStore.StoreImage(plots[0])
		if err != nil {
			client.logger.Warning().
				Error(err).
				Msg("could not store the plot image in the image store")
		} else {
			createAlertRequest.ImageURL = imageLink
		}
	}

	return createAlertRequest
}

func (client *victoropsClient) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message strings.Builder
	desc := stripmd.Strip(trigger.Desc)
	eventsString := client.buildEventsString(events, -1, throttled)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (client *victoropsClient) getMessageType(events moira.NotificationEvents) api.MessageType {
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

func (client *victoropsClient) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := string(state)
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
func (client *victoropsClient) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
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
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(client.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
		if msg := event.CreateMessage(client.location); len(msg) > 0 {
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
