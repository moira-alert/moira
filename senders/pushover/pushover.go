package pushover

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	blackfriday "gopkg.in/russross/blackfriday.v2"

	"github.com/gregdel/pushover"
)

const titleLimit = 250
const urlLimit = 512
const msgLimit = 1024

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
		Message:   sender.buildMessage(events, throttled, trigger),
		Title:     sender.buildTitle(events, trigger),
		Priority:  sender.getMessagePriority(events),
		Retry:     5 * time.Minute,
		Expire:    time.Hour,
		Timestamp: events[len(events)-1].Timestamp,
		HTML:      true,
	}
	url := trigger.GetTriggerURI(sender.frontURI)
	if len(url) < urlLimit {
		pushoverMessage.URL = url
	}
	if len(plot) > 0 {
		reader := bytes.NewReader(plot)
		pushoverMessage.AddAttachment(reader)
	}

	return pushoverMessage
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, throttled bool, trigger moira.TriggerData) string {
	var message strings.Builder

	desc := trigger.Desc
	htmlDesc := string(blackfriday.Run([]byte(desc)))
	htmlDescLen := len([]rune(htmlDesc))
	charsForHTMLTags := htmlDescLen - len([]rune(desc))

	eventsString := sender.buildEventsString(events, -1, throttled)
	eventsStringLen := len([]rune(eventsString))

	descNewLen, eventsNewLen := sender.calculateMessagePartsLength(msgLimit, htmlDescLen, eventsStringLen)

	if htmlDescLen != descNewLen {
		desc = desc[:descNewLen-charsForHTMLTags] + "...\n"
		htmlDesc = string(blackfriday.Run([]byte(desc)))
	}
	if eventsNewLen != eventsStringLen {
		eventsString = sender.buildEventsString(events, eventsNewLen, throttled)
	}

	message.WriteString(htmlDesc)
	message.WriteString(eventsString)
	return message.String()
}

func (sender *Sender) calculateMessagePartsLength(maxChars, descLen, eventsLen int) (descNewLen int, eventsNewLen int) {
	if descLen+eventsLen <= maxChars {
		return descLen, eventsLen
	}
	if descLen > maxChars/2 && eventsLen <= maxChars/2 {
		return maxChars - eventsLen - 10, eventsLen
	}
	if eventsLen > maxChars/2 && descLen <= maxChars/2 {
		return descLen, maxChars - descLen
	}
	return maxChars/2 - 10, maxChars / 2
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
	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s\n", moira.UseString(event.Message))
		} else {
			line += "\n"
		}

		tailStringLen := len([]rune(fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}

	if eventsLenLimitReached {
		eventsString += fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	title := fmt.Sprintf("%s %s %s (%d)", events.GetSubjectState(), trigger.Name, trigger.GetTags(), len(events))
	tags := 1
	for len([]rune(title)) > titleLimit {
		var tagBuffer bytes.Buffer
		for i := 0; i < len(trigger.Tags)-tags; i++ {
			tagBuffer.WriteString(fmt.Sprintf("[%s]", trigger.Tags[i]))
		}
		title = fmt.Sprintf("%s %s %s.... (%d)", events.GetSubjectState(), trigger.Name, tagBuffer.String(), len(events))
		tags++
	}
	return title
}

func (sender *Sender) getMessagePriority(events moira.NotificationEvents) int {
	priority := pushover.PriorityNormal
	for _, event := range events {
		if event.State == moira.StateERROR || event.State == moira.StateEXCEPTION {
			priority = pushover.PriorityEmergency
		}
		if priority != pushover.PriorityEmergency && (event.State == moira.StateWARN || event.State == moira.StateNODATA) {
			priority = pushover.PriorityHigh
		}
	}
	return priority
}
