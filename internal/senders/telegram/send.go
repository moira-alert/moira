package telegram

import (
	"bytes"
	"fmt"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"gopkg.in/tucnak/telebot.v2"
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
func (sender *Sender) SendEvents(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) error {
	msgType := getMessageType(plot)
	message := sender.buildMessage(events, trigger, throttled, characterLimits[msgType])
	sender.logger.Debugf("Calling telegram api with chat_id %s and message body %s", contact.Value, message)
	chat, err := sender.getChat(contact.Value)
	if err != nil {
		return err
	}
	if err := sender.talk(chat, message, plot, msgType); err != nil {
		return fmt.Errorf("failed to send message to telegram contact %s: %s. ", contact.Value, err)
	}
	return nil
}

func (sender *Sender) buildMessage(events moira2.NotificationEvents, trigger moira2.TriggerData, throttled bool, maxChars int) string {
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
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
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
		return nil, fmt.Errorf("can't find recipient %s: %s", uid, err.Error())
	}
	return chat, nil
}

// talk processes one talk
func (sender *Sender) talk(chat *telebot.Chat, message string, plot []byte, messageType messageType) error {
	if messageType == Photo {
		return sender.sendAsPhoto(chat, plot, message)
	}
	return sender.sendAsMessage(chat, message)
}

func (sender *Sender) sendAsMessage(chat *telebot.Chat, message string) error {
	_, err := sender.bot.Send(chat, message)
	if err != nil {
		return fmt.Errorf("can't send event message [%s] to %v: %s", message, chat.ID, err.Error())
	}
	return nil
}

func (sender *Sender) sendAsPhoto(chat *telebot.Chat, plot []byte, caption string) error {
	photo := telebot.Photo{File: telebot.FromReader(bytes.NewReader(plot)), Caption: caption}
	_, err := photo.Send(sender.bot, chat, &telebot.SendOptions{})
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
