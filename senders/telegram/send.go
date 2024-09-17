package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/moira-alert/moira/senders/msgformat"
	"gopkg.in/telebot.v3"

	"github.com/moira-alert/moira"
)

type messageType string

const (
	// Album type used if notification has plots.
	Album messageType = "album"
	// Message type used if notification has no plot.
	Message messageType = "message"
)

const (
	albumCaptionMaxCharacters = 1024
	messageMaxCharacters      = 4096
)

var characterLimits = map[messageType]int{
	Message: messageMaxCharacters,
	Album:   albumCaptionMaxCharacters,
}

const telegramParseMode = telebot.ModeHTML

var unmarshalTypeError *json.UnmarshalTypeError

// Chat is a structure that represents chat metadata required to send message to recipient.
// It implements gopkg.in/telebot.v3#Recipient interface and thus might be passed to telebot methods directly.
type Chat struct {
	ID       int64 `json:"chat_id" example:"-1001234567890"`
	ThreadID int   `json:"thread_id,omitempty" example:"10"`
}

var brokenContactAPIErrors = map[*telebot.Error]struct{}{
	telebot.ErrUnauthorized:         {},
	telebot.ErrUserIsDeactivated:    {},
	telebot.ErrNoRightsToSendPhoto:  {},
	telebot.ErrChatNotFound:         {},
	telebot.ErrNoRightsToSend:       {},
	telebot.ErrKickedFromGroup:      {},
	telebot.ErrBlockedByUser:        {},
	telebot.ErrKickedFromSuperGroup: {},
	telebot.ErrKickedFromChannel:    {},
	telebot.ErrNotStartedByUser:     {},
}

// Recipient allow Chat implements gopkg.in/telebot.v3#Recipient interface.
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
		err = checkBrokenContactError(sender.logger, err)

		return sender.retryIfBadMessageError(err, events, contact, trigger, plots, throttled, chat, msgType)
	}

	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool, maxChars int) string {
	return sender.formatter.Format(msgformat.MessageFormatterParams{
		Events:          events,
		Trigger:         trigger,
		MessageMaxChars: maxChars,
		Throttled:       throttled,
	})
}

func (sender *Sender) getChat(contactValue string) (*Chat, error) {
	var chat *Chat
	var err error

	switch {
	// For private channel contactValue is transformed to be able to fetch it from telegram
	case strings.HasPrefix(contactValue, "%"):
		contactValue = "-100" + contactValue[1:]
		chat, err = sender.getChatFromTelegram(contactValue)
	// For public channel contactValue is transformed to be able to fetch it from telegram
	case strings.HasPrefix(contactValue, "#"):
		contactValue = "@" + contactValue[1:]
		chat, err = sender.getChatFromTelegram(contactValue)
	// For the rest of the cases (private chats, groups, supergroups), Chat data is stored in DB
	default:
		chat, err = sender.getChatFromDb(contactValue)
	}

	return chat, err
}

func (sender *Sender) getChatFromDb(contactValue string) (*Chat, error) {
	chatRaw, err := sender.DataBase.GetChatByUsername(messenger, contactValue)
	if err != nil {
		return nil, fmt.Errorf("failed to get username chat: %w", err)
	}

	chat := &Chat{}
	if err := json.Unmarshal([]byte(chatRaw), chat); err != nil {
		// For Moira < 2.12.0 compatibility
		// Before 2.12.0 `moira-telegram-users:user` only stored telegram channel IDs
		// After 2.12.0 `moira-telegram-users:user` stores Chat structure
		if errors.As(err, &unmarshalTypeError) {
			chatID, parseErr := strconv.ParseInt(chatRaw, 10, 64)
			if parseErr != nil {
				return nil, fmt.Errorf("failed to parse chatRaw: %s as int64: %w", chatRaw, parseErr)
			}

			return &Chat{
				ID: chatID,
			}, nil
		}

		return nil, fmt.Errorf("failed to unmarshal chat data %s: %w", chatRaw, err)
	}

	return chat, nil
}

func (sender *Sender) getChatFromTelegram(username string) (*Chat, error) {
	telegramChat, err := sender.bot.ChatByUsername(username)
	if err != nil {
		err = sender.removeTokenFromError(err)
		return nil, fmt.Errorf("can't find recipient %s: %w", username, err)
	}

	chat := Chat{
		ID: telegramChat.ID,
	}

	return &chat, nil
}

func (sender *Sender) setChat(message *telebot.Message) error {
	contactValue, err := sender.getContactValueByMessage(message)
	if err != nil {
		return fmt.Errorf("failed to get contact value from message: %w", err)
	}

	chat := &Chat{
		ID:       message.Chat.ID,
		ThreadID: message.ThreadID,
	}

	chatRaw, err := json.Marshal(chat)
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}

	if err = sender.DataBase.SetUsernameChat(messenger, contactValue, string(chatRaw)); err != nil {
		return fmt.Errorf("failed to set username chat: %w", err)
	}

	return nil
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
	_, err := sender.bot.Send(chat, message, &telebot.SendOptions{
		ThreadID:              chat.ThreadID,
		ParseMode:             telegramParseMode,
		DisableWebPagePreview: true,
	})
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

	_, err := sender.bot.SendAlbum(chat, album, &telebot.SendOptions{
		ThreadID:              chat.ThreadID,
		ParseMode:             telegramParseMode,
		DisableWebPagePreview: true,
	})
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

func (sender *Sender) retryIfBadMessageError(
	err error,
	events []moira.NotificationEvent,
	contact moira.ContactData,
	trigger moira.TriggerData,
	plots [][]byte,
	throttled bool,
	chat *Chat,
	msgType messageType,
) error {
	var e moira.SenderBrokenContactError
	if isBrokenContactErr := errors.As(err, &e); !isBrokenContactErr {
		if _, isBadMessage := checkBadMessageError(err); isBadMessage {
			// There are some problems with message formatting.
			// For example, it is too long, or have unsupported tags and so on.
			// Events should not be lost, so retry to send it without description.

			sender.logger.Warning().
				String(moira.LogFieldNameContactID, contact.ID).
				String(moira.LogFieldNameContactType, contact.Type).
				String(moira.LogFieldNameContactValue, contact.Value).
				String(moira.LogFieldNameTriggerID, trigger.ID).
				String(moira.LogFieldNameTriggerName, trigger.Name).
				Error(err).
				Msg("Failed to send alert because of bad description. Retrying now.")

			trigger.Desc = badFormatMessage
			message := sender.buildMessage(events, trigger, throttled, characterLimits[msgType])

			err = sender.talk(chat, message, plots, msgType)
			return checkBrokenContactError(sender.logger, err)
		}
	}

	return err
}

var badMessageFormatErrors = map[*telebot.Error]struct{}{
	telebot.ErrTooLarge:       {},
	telebot.ErrTooLongMessage: {},
}

const (
	errMsgPrefixCannotParseInputMedia = "telegram: Bad Request: can't parse InputMedia: Can't parse entities: Unsupported start tag"
	errMsgPrefixCaptionTooLong        = "telegram: Bad Request: message caption is too long (400)"
	errMsgPrefixCannotParseEntities   = "telegram: Bad Request: can't parse entities: Unsupported start tag"
)

func checkBadMessageError(err error) (error, bool) {
	if err == nil {
		return nil, false
	}

	var telebotErr *telebot.Error
	if ok := errors.As(err, &telebotErr); ok {
		if isBadMessageFormatError(telebotErr) {
			return telebotErr, true
		}
	}

	errMsg := err.Error()
	if strings.HasPrefix(errMsg, errMsgPrefixCannotParseInputMedia) ||
		strings.HasPrefix(errMsg, errMsgPrefixCaptionTooLong) ||
		strings.HasPrefix(errMsg, errMsgPrefixCannotParseEntities) {
		return err, true
	}

	return err, false
}

func isBadMessageFormatError(e *telebot.Error) bool {
	_, exists := badMessageFormatErrors[e]
	return exists
}
