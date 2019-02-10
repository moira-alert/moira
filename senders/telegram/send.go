package telegram

import (
	"bytes"
	"fmt"

	"gopkg.in/tucnak/telebot.v2"

	"github.com/moira-alert/moira"
)

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	sender.logger.Debugf("Calling telegram api with chat_id %s and message body %s", contact.Value, message)
	if err := sender.talk(contact.Value, message, plot); err != nil {
		return fmt.Errorf("failed to send message to telegram contact %s: %s. ", contact.Value, err)
	}
	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var buffer bytes.Buffer
	state := events.GetSubjectState()
	tags := trigger.GetTags()
	emoji := emojiStates[state]

	buffer.WriteString(fmt.Sprintf("%s%s %s %s (%d)\n", emoji, state, trigger.Name, tags, len(events)))

	messageLimitReached := false
	lineCount := 0

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		if buffer.Len()+len(line) > telegramMessageLimit-400 {
			messageLimitReached = true
			break
		}
		buffer.WriteString(line)
		lineCount++
	}

	if messageLimitReached {
		buffer.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-lineCount))
	}
  url := trigger.GetTriggerUri(sender.frontURI)
  if url != "" {
		buffer.WriteString(fmt.Sprintf("\n\n%s\n", url))
	}

	if throttled {
		buffer.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}
	return buffer.String()
}

// talk processes one talk
func (sender *Sender) talk(username, message string, plot []byte) error {
	var err error
	uid, err := sender.DataBase.GetIDByUsername(messenger, username)
	if err != nil {
		return fmt.Errorf("failed to get username uuid: %s", err.Error())
	}
	chat, err := sender.bot.ChatByID(uid)
	if err != nil {
		return fmt.Errorf("can't find recepient %s: %s", uid, err.Error())
	}
	postedMessage, err := sender.bot.Send(chat, message)
	if err != nil {
		return fmt.Errorf("can't send event message [%s] to %s: %s", message, uid, err.Error())
	}
	if len(plot) > 0 {
		photo := telebot.Photo{File: telebot.FromReader(bytes.NewReader(plot))}
		_, err = photo.Send(sender.bot, chat, &telebot.SendOptions{ReplyTo: postedMessage})
		if err != nil {
			sender.logger.Errorf("can't send event plot to %s: %s", uid, err.Error())
		}
	}
	return nil
}
