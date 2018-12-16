package pushover

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"

	"github.com/gregdel/pushover"
)

// Sender implements moira sender interface via pushover
type Sender struct {
	APIToken string
	FrontURI string
	log      moira.Logger
	location *time.Location
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger,
	location *time.Location, dateTimeFormat string) error {

	sender.APIToken = senderSettings["api_token"]
	if sender.APIToken == "" {
		return fmt.Errorf("Can not read pushover api_token from config")
	}
	sender.log = logger
	sender.FrontURI = senderSettings["front_uri"]
	sender.location = location
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData,
	trigger moira.TriggerData, plot []byte, throttled bool) error {

	api := pushover.New(sender.APIToken)
	recipient := pushover.NewRecipient(contact.Value)

	subjectState := events.GetSubjectState()
	title := fmt.Sprintf("%s %s %s (%d)", subjectState, trigger.Name, trigger.GetTags(), len(events))
	timestamp := events[len(events)-1].Timestamp

	var message bytes.Buffer
	priority := pushover.PriorityNormal
	for i, event := range events {
		if i > 4 {
			break
		}
		if event.State == "ERROR" || event.State == "EXCEPTION" {
			priority = pushover.PriorityEmergency
		}
		if priority != pushover.PriorityEmergency && (event.State == "WARN" || event.State == "NODATA") {
			priority = pushover.PriorityHigh
		}
		value := strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64)
		message.WriteString(fmt.Sprintf("%s: %s = %s (%s to %s)",
			time.Unix(event.Timestamp, 0).In(sender.location).Format("15:04"),
			event.Metric, value, event.OldState, event.State))
		if len(moira.UseString(event.Message)) > 0 {
			message.WriteString(fmt.Sprintf(". %s\n", moira.UseString(event.Message)))
		} else {
			message.WriteString("\n")
		}
	}

	if len(events) > 5 {
		message.WriteString(fmt.Sprintf("\n...and %d more events.", len(events)-5))
	}

	if throttled {
		message.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}

	sender.log.Debugf("Calling pushover with message title %s, body %s", title, message.String())

	pushoverMessage := &pushover.Message{
		Message:   message.String(),
		Title:     title,
		Priority:  priority,
		Retry:     5 * time.Minute,
		Expire:    time.Hour,
		Timestamp: timestamp,
		URL:       fmt.Sprintf("%s/trigger/%s", sender.FrontURI, events[0].TriggerID),
	}
	_, err := api.SendMessage(pushoverMessage, recipient)
	if err != nil {
		return fmt.Errorf("Failed to send message to pushover user %s: %s", contact.Value, err.Error())
	}
	return nil
}

// GetLocation implements Sender interface GetLocation
func (sender *Sender) GetLocation() *time.Location {
	return sender.location
}
