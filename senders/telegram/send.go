package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/telebot.v3"

	"github.com/moira-alert/moira"
)

type messageType string

const (
	// Album type used if notification has plots.
	Album messageType = "album"
	// Message type used if notification has not plot.
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

type Chat struct {
	ID       int64            `json:"chatId" example:"-1001234567890"`
	Type     telebot.ChatType `json:"type" example:"supergroup"`
	ThreadID int              `json:"threadId,omitempty" example:"10"`
}

var brokenContactAPIErrors = map[*telebot.Error]bool{
	telebot.ErrUnauthorized:         true,
	telebot.ErrUserIsDeactivated:    true,
	telebot.ErrNoRightsToSendPhoto:  true,
	telebot.ErrChatNotFound:         true,
	telebot.ErrNoRightsToSend:       true,
	telebot.ErrKickedFromGroup:      true,
	telebot.ErrBlockedByUser:        true,
	telebot.ErrKickedFromSuperGroup: true,
	telebot.ErrKickedFromChannel:    true,
	telebot.ErrNotStartedByUser:     true,
}

// Chat implements gopkg.in/telebot.v3#Recipient interface.
func (c *Chat) Recipient() string {
	return strconv.FormatInt(c.ID, 10)
}

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	msgType := getMessageType(plots)
	message := sender.buildMessage(events, trigger, throttled, characterLimits[msgType])
	sender.logger.Debug().
		String("chat_id", contact.Value).
		String("message", message).
		Msg("Calling telegram api")

	chat, err := sender.getChat(contact.Value)
	if err != nil {
		return checkBrokenContactError(sender.logger, err)
	}
	if err := sender.talk(chat, message, plots, msgType); err != nil {
		return checkBrokenContactError(sender.logger, err)
	}
	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool, maxChars int) string {
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
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
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

func (sender *Sender) getChat(contactValue string) (*Chat, error) {
	var chat *Chat
	var err error

	switch {
	// for private channel contactValue is transformed to be able to fetch it from telegram
	case strings.HasPrefix(contactValue, "%"):
		contactValue = "-100" + contactValue[1:]
		chat, err = sender.getChatFromTelegram(contactValue)
	// for public channel contactValue is transformed to be able to fetch it from telegram
	case strings.HasPrefix(contactValue, "#"):
		contactValue = "@" + contactValue[1:]
		chat, err = sender.getChatFromTelegram(contactValue)
	// for the rest of the cases (private chats, groups, supergroups), Chat data is stored in DB.
	default:
		chat, err = sender.getChatFromDb(contactValue)
	}

	return chat, err
}

func (sender *Sender) getChatFromDb(contactValue string) (*Chat, error) {
	var err error

	chatRaw, err := sender.DataBase.GetIDByUsername(messenger, contactValue)
	if err != nil {
		return nil, fmt.Errorf("failed to get username uuid: %s", err.Error())
	}

	chat := Chat{}
	err = json.Unmarshal([]byte(chatRaw), &chat)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat data %s: %s", chatRaw, err.Error())
	}
	return &chat, nil
}

func (sender *Sender) getChatFromTelegram(username string) (*Chat, error) {
	var err error

	telegramChat, err := sender.bot.ChatByUsername(username)
	if err != nil {
		err = sender.removeTokenFromError(err)
		return nil, fmt.Errorf("can't find recipient %s: %s", username, err.Error())
	}

	chat := Chat{
		Type: telegramChat.Type,
		ID:   telegramChat.ID,
	}
	return &chat, nil
}

func (sender *Sender) setChat(message *telebot.Message) (*Chat, error) {
	contactValue, err := sender.getContactValueByMessage(message)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact value from message: %s", err.Error())
	}

	chat := &Chat{
		Type:     message.Chat.Type,
		ID:       message.Chat.ID,
		ThreadID: message.ThreadID,
	}

	chatString, err := json.Marshal(chat)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat: %s", err.Error())
	}

	err = sender.DataBase.SetUsernameID(messenger, contactValue, string(chatString))
	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (sender *Sender) getContactValueByMessage(message *telebot.Message) (string, error) {
	var contactValue string
	var err error

	switch {
	case message.Chat.Type == telebot.ChatPrivate:
		contactValue = "@" + message.Chat.Username
	case message.Chat.Type == telebot.ChatSuperGroup && message.ThreadID != 0:
		contactValue = fmt.Sprintf("%d/%d", message.Chat.ID, message.ThreadID)
	case message.Chat.Type == telebot.ChatSuperGroup || message.Chat.Type == telebot.ChatGroup:
		contactValue = message.Chat.Title
	case message.Chat.Type == telebot.ChatChannel:
		contactValue = "#" + message.Chat.Username
	case message.Chat.Type == telebot.ChatChannelPrivate:
		contactValue = strings.Replace(message.Chat.Recipient(), "-100", "%", -1)
	default:
		err = fmt.Errorf("unknown chat type")
	}

	return contactValue, err
}

// talk processes one talk.
func (sender *Sender) talk(chat *Chat, message string, plots [][]byte, messageType messageType) error {
	if messageType == Album {
		sender.logger.Debug().Msg("talk as album")
		return sender.sendAsAlbum(chat, plots, message)
	}
	sender.logger.Debug().Msg("talk as send message")
	return sender.sendAsMessage(chat, message)
}

func (sender *Sender) sendAsMessage(chat *Chat, message string) error {
	_, err := sender.bot.Send(chat, message, &telebot.SendOptions{ThreadID: chat.ThreadID})
	if err != nil {
		err = sender.removeTokenFromError(err)
		sender.logger.Debug().
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

	var e *telebot.Error
	if ok := errors.As(err, &e); ok {
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

func isBrokenContactAPIError(err *telebot.Error) bool {
	_, exists := brokenContactAPIErrors[err]
	return exists
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

func (sender *Sender) sendAsAlbum(chat *Chat, plots [][]byte, caption string) error {
	album := prepareAlbum(plots, caption)

	_, err := sender.bot.SendAlbum(chat, album, &telebot.SendOptions{ThreadID: chat.ThreadID})
	if err != nil {
		err = sender.removeTokenFromError(err)
		sender.logger.Debug().
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
