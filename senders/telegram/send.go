package telegram

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

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
	photoCaptionMaxCharacters = 1024
	messageMaxCharacters      = 4096
)

var characterLimits = map[messageType]int{
	Message: messageMaxCharacters,
	Photo:   photoCaptionMaxCharacters,
}

var (
	mdHeaderRegex = regexp.MustCompile(`(?m)^\s*#{1,}\s*(?P<headertext>[^#\n]+)$`)
)

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
	var buffer strings.Builder

	state := events.GetSubjectState()
	tags := trigger.GetTags()
	emoji := emojiStates[state]
	title := fmt.Sprintf("%s%s %s %s (%d)\n", emoji, state, trigger.Name, tags, len(events))
	titleLen := len([]rune(title))

	desc := trigger.Desc
	if trigger.Desc != "" {
		// Replace MD headers (## header text) with **header text** that telegram supports
		desc = mdHeaderRegex.ReplaceAllString(trigger.Desc, "**$headertext**")
		desc += "\n"
	}
	descLen := len([]rune(desc))

	eventsString := sender.buildEventsString(events, -1, throttled, trigger)
	eventsStringLen := len([]rune(eventsString))

	if titleLen+descLen+eventsStringLen <= messageMaxCharacters {
		buffer.WriteString(title)
		buffer.WriteString(desc)
		buffer.WriteString(eventsString)
		return buffer.String()
	}

	charsLeftAfterTitle := messageMaxCharacters - titleLen
	if descLen > charsLeftAfterTitle/2 && eventsStringLen > charsLeftAfterTitle/2 {
		// Trim both desc and events string to half the charsLeftAfter title
		desc = desc[:charsLeftAfterTitle/2-10] + "...\n"
		eventsString = sender.buildEventsString(events, charsLeftAfterTitle/2, throttled, trigger)

	} else if descLen > charsLeftAfterTitle/2 {
		// Trim the desc to the chars left after using the whole events string
		charsForDesc := charsLeftAfterTitle - eventsStringLen
		desc = desc[:charsForDesc-10] + "...\n"

	} else if eventsStringLen > charsLeftAfterTitle/2 {
		// Trim the events string to the chars left after using the whole desc
		charsForEvents := charsLeftAfterTitle - descLen
		eventsString = sender.buildEventsString(events, charsForEvents, throttled, trigger)

	} else {
		desc = desc[:charsLeftAfterTitle/2-10] + "...\n"
		eventsString = sender.buildEventsString(events, charsLeftAfterTitle/2, throttled, trigger)

	}
	buffer.WriteString(title)
	buffer.WriteString(desc)
	buffer.WriteString(eventsString)
	return buffer.String()
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if n is negative buildEventsString does not limit the events string
func (sender *Sender) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool, trigger moira.TriggerData) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, fix your system or tune this trigger to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}

	var urlString string
	url := trigger.GetTriggerURI(sender.frontURI)
	if url != "" {
		urlString = fmt.Sprintf("\n\n%s\n", url)
	}
	charsLeftForEvents := charsForEvents - len([]rune(urlString)) - charsForThrottleMsg

	var eventsString string
	var tailString string
	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		tailString = fmt.Sprintf("\n\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune(tailString))
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
	if url != "" {
		eventsString += urlString
	}
	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
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
		return sender.sendAsPhoto(chat, plot, message)
	}
	return sender.sendAsMessage(chat, message)
}

func (sender *Sender) sendAsMessage(chat *telebot.Chat, message string) error {
	_, err := sender.bot.Send(chat, message, []interface{}{telebot.ModeMarkdown})
	if err != nil {
		return fmt.Errorf("can't send event message [%s] to %v: %s", message, chat.ID, err.Error())
	}
	return nil
}

func (sender *Sender) sendAsPhoto(chat *telebot.Chat, plot []byte, caption string) error {
	photo := telebot.Photo{File: telebot.FromReader(bytes.NewReader(plot)), Caption: caption}
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
