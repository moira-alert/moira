package telegram

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/skbkontur/bot"

	"github.com/moira-alert/moira-alert"
)

var (
	api                  bot.Bot
	telegramMessageLimit = 4096
	emojiStates          = map[string]string{
		"OK":     "\xe2\x9c\x85",
		"WARN":   "\xe2\x9a\xa0",
		"ERROR":  "\xe2\xad\x95",
		"NODATA": "\xf0\x9f\x92\xa3",
		"TEST":   "\xf0\x9f\x98\x8a",
	}
)

// Sender implements moira sender interface via telegram
type Sender struct {
	DB       bot.Database
	APIToken string
	FrontURI string
	log      moira.Logger
}

//Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger) error {
	sender.APIToken = senderSettings["api_token"]
	if sender.APIToken == "" {
		return fmt.Errorf("Can not read telegram api_token from config")
	}
	sender.log = logger
	sender.FrontURI = senderSettings["front_uri"]

	var err error
	api, err = bot.StartTelebot(sender.APIToken, sender.DB)
	if err != nil {
		sender.log.Errorf("Error starting bot: %s", err)
	}
	return nil
}

//SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.EventsData, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {

	var message bytes.Buffer

	state := events.GetSubjectState()
	tags := trigger.GetTags()

	emoji := emojiStates[state]
	message.WriteString(fmt.Sprintf("%s%s %s %s (%d)\n", emoji, state, trigger.Name, tags, len(events)))

	messageLimitReached := false
	lineCount := 0

	for _, event := range events {
		value := strconv.FormatFloat(event.Value, 'f', -1, 64)
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", time.Unix(event.Timestamp, 0).Format("15:04"), event.Metric, value, event.OldState, event.State)
		if len(event.Message) > 0 {
			line += fmt.Sprintf(". %s", event.Message)
		}
		if message.Len()+len(line) > telegramMessageLimit-400 {
			messageLimitReached = true
			break
		}
		message.WriteString(line)
		lineCount++
	}

	if messageLimitReached {
		message.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-lineCount))
	}

	message.WriteString(fmt.Sprintf("\n\n%s/#/events/%s\n", sender.FrontURI, events[0].TriggerID))

	if throttled {
		message.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}

	sender.log.Debugf("Calling telegram api with chat_id %s and message body %s", contact.Value, message.String())

	if err := api.Talk(contact.Value, message.String()); err != nil {
		return fmt.Errorf("Failed to send message to telegram contact %s: %s. ", contact.Value, err)
	}
	return nil

}
