package telegram

import (
	"bytes"
	"fmt"
	"github.com/moira-alert/moira/senders"
	"gopkg.in/tucnak/telebot.v2"
	"strings"

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
	photoCaptionMaxCharacters = 1024
	messageMaxCharacters      = 4096
)

var characterLimits = map[messageType]int{
	Message: messageMaxCharacters,
	Photo:   photoCaptionMaxCharacters,
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
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

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool, maxChars int) string {
	var message strings.Builder

	title := senders.BuildTitle(events, trigger, sender.frontURI)
	titleLen := len([]rune(title))

	desc := sender.buildDescription(trigger)
	descLen := len([]rune(desc))

	eventsString := senders.BuildEventsString(events, -1, throttled, sender.location)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := messageMaxCharacters - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)

	if descLen != descNewLen {
		desc = desc[:descNewLen] + "...\n"
	}
	if eventsNewLen != eventsStringLen {
		eventsString = senders.BuildEventsString(events, eventsNewLen, throttled, sender.location)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (sender *Sender) buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc = trigger.Desc
		desc += "\n"
	}
	return desc
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
