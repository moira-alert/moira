package pushover

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"

	"github.com/gregdel/pushover"
)

const printEventsCount int = 5

// Sender implements moira sender interface via pushover
type Sender struct {
	logger   moira.Logger
	location *time.Location
	client   *pushover.Pushover

	apiToken string
	frontURI string
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {

	sender.apiToken = senderSettings["api_token"]
	if sender.apiToken == "" {
		return fmt.Errorf("can not read pushover api_token from config")
	}
	sender.client = pushover.New(sender.apiToken)
	sender.logger = logger
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location
	return nil
}

// SendEvents implements pushover build and send message functionality
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	pushoverMessage := sender.makePushoverMessage(events, contact, trigger, plot, throttled)

	sender.logger.Debugf("Calling pushover with message title %s, body %s", pushoverMessage.Title, pushoverMessage.Message)
	recipient := pushover.NewRecipient(contact.Value)
	_, err := sender.client.SendMessage(pushoverMessage, recipient)
	if err != nil {
		return fmt.Errorf("failed to send %s event message to pushover user %s: %s", trigger.ID, contact.Value, err.Error())
	}
	return nil
}

func (sender *Sender) makePushoverMessage(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) *pushover.Message {
	pushoverMessage := &pushover.Message{
		Message:   sender.buildMessage(events, throttled),
		Title:     sender.buildTitle(events, trigger),
		Priority:  sender.getMessagePriority(events),
		URL:       sender.buildTriggerURL(trigger),
		Retry:     5 * time.Minute,
		Expire:    time.Hour,
		Timestamp: events[len(events)-1].Timestamp,
	}

	if len(plot) > 0 {
		reader := bytes.NewReader(plot)
		if err := pushoverMessage.AddAttachment(reader); err != nil {
			sender.logger.Errorf("Failed to send %s event plot to pushover user %s: %s", trigger.ID, contact.Value, err.Error())
		}
	}

	return pushoverMessage
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, throttled bool) string {
	var message bytes.Buffer
	for i, event := range events {
		if i > printEventsCount-1 {
			break
		}
		value := strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64)
		timeStr := time.Unix(event.Timestamp, 0).In(sender.location).Format("15:04")
		message.WriteString(fmt.Sprintf("%s: %s = %s (%s to %s)", timeStr, event.Metric, value, event.OldState, event.State))
		if len(moira.UseString(event.Message)) > 0 {
			message.WriteString(fmt.Sprintf(". %s\n", moira.UseString(event.Message)))
		} else {
			message.WriteString("\n")
		}
	}
	if len(events) > printEventsCount {
		message.WriteString(fmt.Sprintf("\n...and %d more events.", len(events)-printEventsCount))
	}

	if throttled {
		message.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}
	return message.String()
}

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	subjectState := events.GetSubjectState()
	return fmt.Sprintf("%s %s %s (%d)", subjectState, trigger.Name, trigger.GetTags(), len(events))
}

func (sender *Sender) getMessagePriority(events moira.NotificationEvents) int {
	priority := pushover.PriorityNormal
	for _, event := range events {
		if event.State == "ERROR" || event.State == "EXCEPTION" {
			priority = pushover.PriorityEmergency
		}
		if priority != pushover.PriorityEmergency && (event.State == "WARN" || event.State == "NODATA") {
			priority = pushover.PriorityHigh
		}
	}
	return priority
}

func (sender *Sender) buildTriggerURL(trigger moira.TriggerData) string {
	return fmt.Sprintf("%s/trigger/%s", sender.frontURI, trigger.ID)
}
