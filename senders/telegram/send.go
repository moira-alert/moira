package telegram

import (
	"bytes"
	"fmt"

	"gopkg.in/tucnak/telebot.v2"

	"github.com/moira-alert/moira"
)

type messageType string

const (
	// Photo type used if notification has plot
	Photo messageType = "photo"
	// Message type used if notification has not plot
	Message messageType = "message"
)

const (
	photoCaptionMaxCharacters     = 1024
	messageMaxCharacters          = 4096
	additionalInfoCharactersCount = 400
)

var characterLimits = map[messageType]int{
	Message: messageMaxCharacters,
	Photo:   photoCaptionMaxCharacters,
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	messageType := getMessageType(plot)
	message := sender.buildMessage(events, trigger, throttled, characterLimits[messageType])
	sender.logger.Debugf("Calling telegram api with chat_id %s and message body %s", contact.Value, message)
	chat, err := sender.getChat(contact.Value)
	if err != nil {
		return err
	}
	if err := sender.talk(chat, message, plot, messageType); err != nil {
		return fmt.Errorf("failed to send message to telegram contact %s: %s. ", contact.Value, err)
	}
	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool, maxChars int) string {
	var buffer bytes.Buffer
	state := events.GetSubjectState()
	tags := trigger.GetTags()
	emoji := emojiStates[state]

	title := fmt.Sprintf("%s%s %s %s (%d)\n", emoji, state, trigger.Name, tags, len(events))
	buffer.WriteString(title)

	var messageCharsCount, printEventsCount int
	messageCharsCount += len([]rune(title))
	messageLimitReached := false

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		lineCharsCount := len([]rune(line))
		if messageCharsCount+lineCharsCount > maxChars-additionalInfoCharactersCount {
			messageLimitReached = true
			break
		}
		buffer.WriteString(line)
		messageCharsCount += lineCharsCount
		printEventsCount++
	}

	if messageLimitReached {
		buffer.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-printEventsCount))
	}
  url := trigger.GetTriggerURI(sender.frontURI)
  if url != "" {
		buffer.WriteString(fmt.Sprintf("\n\n%s\n", url))
	}

	if throttled {
		buffer.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}
	return buffer.String()
}

func (sender *Sender) getChat(username string) (*telebot.Chat, error) {
	uid, err := sender.DataBase.GetIDByUsername(messenger, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get username uuid: %s", err.Error())
	}
	chat, err := sender.bot.ChatByID(uid)
	if err != nil {
		return nil, fmt.Errorf("can't find recepient %s: %s", uid, err.Error())
	}
	return chat, nil
}

// talk processes one talk
func (sender *Sender) talk(chat *telebot.Chat, message string, plot []byte, messageType messageType) error {
	if messageType == Photo {
		return sender.sendPhoto(chat, plot, message)
	}
	return sender.sendMessage(chat, message)
}

func (sender *Sender) sendMessage(chat *telebot.Chat, message string) error {
	_, err := sender.bot.Send(chat, message)
	if err != nil {
		return fmt.Errorf("can't send event message [%s] to %v: %s", message, chat.ID, err.Error())
	}
	return nil
}

func (sender *Sender) sendPhoto(chat *telebot.Chat, plot []byte, caption string) error {
	photo := telebot.Photo{File: telebot.FromReader(bytes.NewReader(plot)), Caption: caption, Width: 800, Height: 400}
	_, err := photo.Send(sender.bot, chat, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
	if err != nil {
		return fmt.Errorf("can't send event plot to %v: %s", chat.ID, err.Error())
	}
	return nil
}

func getMessageType(plot []byte) messageType {
	if len(plot) > 0 {
		return Photo
	}
	return Message
}
