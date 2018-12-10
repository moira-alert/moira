package telegram

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
)

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData,
	trigger moira.TriggerData, plot []byte, throttled bool) error {

	var message bytes.Buffer
	state := events.GetSubjectState()
	tags := trigger.GetTags()
	emoji := emojiStates[state]

	message.WriteString(fmt.Sprintf("%s%s %s %s (%d)\n", emoji, state, trigger.Name, tags, len(events)))

	messageLimitReached := false
	lineCount := 0

	for _, event := range events {
		value := strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64)
		eventTime := time.Unix(event.Timestamp, 0).In(sender.location)
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", eventTime.Format("15:04"),
			event.Metric, value, event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
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

	message.WriteString(fmt.Sprintf("\n\n%s/trigger/%s\n", sender.FrontURI, events[0].TriggerID))

	if throttled {
		message.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}

	sender.logger.Debugf("Calling telegram api with chat_id %s and message body %s", contact.Value, message.String())

	if err := sender.talk(contact.Value, message.String()); err != nil {
		return fmt.Errorf("Failed to send message to telegram contact %s: %s. ", contact.Value, err)
	}
	return nil
}

// talk processes one talk
func (sender *Sender) talk(username, message string) error {
	var err error
	uid, err := sender.DataBase.GetIDByUsername(messenger, username)
	if err != nil {
		return fmt.Errorf("failed to get username uuid: %s", err.Error())
	}
	chat, err := sender.bot.ChatByID(uid)
	if err != nil {
		return fmt.Errorf("can't find recepient %s: %s", uid, err.Error())
	}
	_, err = sender.bot.Send(chat, message)
	if err != nil {
		return fmt.Errorf("can't send message [%s] to %s: %s", message, uid, err.Error())
	}
	return nil
}
