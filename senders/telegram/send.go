package telegram

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/tucnak/telebot.v2"

	"github.com/moira-alert/moira"
)

type messageType string

const (
	// Album type used if notification has plots
	Album messageType = "album"
	// Message type used if notification has not plot
	Message messageType = "message"
)

const (
	albumCaptionMaxCharacters     = 1024
	messageMaxCharacters          = 4096
	additionalInfoCharactersCount = 400
)

var characterLimits = map[messageType]int{
	Message: messageMaxCharacters,
	Album:   albumCaptionMaxCharacters,
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	client, ok := sender.clients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	msgType := getMessageType(plots)
	message := client.buildMessage(events, trigger, throttled, characterLimits[msgType])
	client.logger.Debug().
		String("chat_id", contact.Value).
		String("message", message).
		Msg("Calling telegram api")

	chat, err := client.getChat(contact.Value)
	if err != nil {
		return checkBrokenContactError(client.logger, err)
	}

	if err := client.talk(chat, message, plots, msgType); err != nil {
		return checkBrokenContactError(client.logger, err)
	}

	return nil
}

func (client *telegramClient) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool, maxChars int) string {
	var buffer bytes.Buffer
	state := events.GetCurrentState(throttled)
	tags := trigger.GetTags()
	emoji := emojiStates[state]

	title := fmt.Sprintf("%s%s %s %s (%d)\n", emoji, state, trigger.Name, tags, len(events))
	buffer.WriteString(title)

	var messageCharsCount, printEventsCount int
	messageCharsCount += len([]rune(title))
	messageLimitReached := false

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(client.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
		if msg := event.CreateMessage(client.location); len(msg) > 0 {
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
	url := trigger.GetTriggerURI(client.frontURI)
	if url != "" {
		buffer.WriteString(fmt.Sprintf("\n\n%s\n", url))
	}

	if throttled {
		buffer.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}

	return buffer.String()
}

func (client *telegramClient) getChatUID(username string) (string, error) {
	var uid string
	if strings.HasPrefix(username, "%") {
		uid = "-100" + username[1:]
	} else {
		var err error
		uid, err = client.database.GetIDByUsername(messenger, username)
		if err != nil {
			return "", fmt.Errorf("failed to get username uuid: %s", err.Error())
		}
	}

	return uid, nil
}

func (client *telegramClient) getChat(username string) (*telebot.Chat, error) {
	uid, err := client.getChatUID(username)
	if err != nil {
		return nil, err
	}

	chat, err := client.bot.ChatByID(uid)
	if err != nil {
		err = removeTokenFromError(err, client.bot)
		return nil, fmt.Errorf("can't find recipient %s: %s", uid, err.Error())
	}

	return chat, nil
}

// talk processes one talk
func (client *telegramClient) talk(chat *telebot.Chat, message string, plots [][]byte, messageType messageType) error {
	if messageType == Album {
		client.logger.Debug().Msg("talk as album")
		return client.sendAsAlbum(chat, plots, message)
	}

	client.logger.Debug().Msg("talk as send message")
	return client.sendAsMessage(chat, message)
}

func (client *telegramClient) sendAsMessage(chat *telebot.Chat, message string) error {
	_, err := client.bot.Send(chat, message)
	if err != nil {
		err = removeTokenFromError(err, client.bot)
		client.logger.Debug().
			String("message", message).
			Int64("chat_id", chat.ID).
			Error(err).
			Msg("Can't send event message to telegram")
	}

	return err
}

func checkBrokenContactError(logger moira.Logger, err error) error {
	logger.Debug().Msg("Check broken contact")
	if err == nil {
		return nil
	}

	if e, ok := err.(*telebot.APIError); ok {
		logger.Debug().
			Int("code", e.Code).
			String("msg", e.Message).
			String("desc", e.Description).
			Msg("It's telebot.APIError from talk()")

		if isBrokenContactAPIError(e) {
			return moira.NewSenderBrokenContactError(err)
		}
	}

	if strings.HasPrefix(err.Error(), "failed to get username uuid") {
		logger.Debug().
			Error(err).
			Msg("It's error from getChat()")
		return moira.NewSenderBrokenContactError(err)
	}

	return err
}

func isBrokenContactAPIError(err *telebot.APIError) bool {
	if err.Code == telebot.ErrUnauthorized.Code {
		return true
	}

	if err.Code == telebot.ErrNoRightsToSendPhoto.Code &&
		(err.Description == telebot.ErrNoRightsToSendPhoto.Description ||
			err.Description == telebot.ErrChatNotFound.Description ||
			err.Description == telebot.ErrNoRightsToSend.Description) {
		return true
	}

	if err.Code == telebot.ErrBotKickedFromGroup.Code &&
		(err.Description == telebot.ErrBotKickedFromGroup.Description ||
			err.Description == telebot.ErrBotKickedFromSuperGroup.Description) {
		return true
	}

	return false
}

func prepareAlbum(plots [][]byte, caption string) telebot.Album {
	var album telebot.Album
	for _, plot := range plots {
		photo := &telebot.Photo{File: telebot.FromReader(bytes.NewReader(plot)), Caption: caption}
		album = append(album, photo)
		caption = "" // Caption should be defined only for first photo
	}

	return album
}

func (client *telegramClient) sendAsAlbum(chat *telebot.Chat, plots [][]byte, caption string) error {
	album := prepareAlbum(plots, caption)

	_, err := client.bot.SendAlbum(chat, album)
	if err != nil {
		err = removeTokenFromError(err, client.bot)
		client.logger.Debug().
			Int64("chat_id", chat.ID).
			Error(err).
			Msg("Can't send event plots to telegram chat")
	}

	return err
}

func getMessageType(plots [][]byte) messageType {
	if len(plots) > 0 {
		return Album
	}
	return Message
}
