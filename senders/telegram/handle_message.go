package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/tucnak/telebot.v2"
)

// handleMessage handles incoming messages to start sending events to subscribers chats
func (client *telegramClient) handleMessage(message *telebot.Message) error {
	responseMessage, err := client.getResponseMessage(message)
	if err != nil {
		return err
	}

	if responseMessage != "" {
		if _, err = client.bot.Send(message.Chat, responseMessage); err != nil {
			return removeTokenFromError(err, client.bot)
		}
	}

	return nil
}

func (client *telegramClient) getResponseMessage(message *telebot.Message) (string, error) {
	chatID := strconv.FormatInt(message.Chat.ID, 10)
	switch {
	case message.Chat.Type == telebot.ChatPrivate && message.Text == "/start":
		if message.Chat.Username == "" {
			return "Username is empty. Please add username in Telegram.", nil
		}

		err := client.database.SetUsernameID(messenger, "@"+message.Chat.Username, chatID)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("Okay, %s, your id is %s", strings.Trim(fmt.Sprintf("%s %s", message.Sender.FirstName, message.Sender.LastName), " "), chatID), nil
	case message.Chat.Type == telebot.ChatSuperGroup || message.Chat.Type == telebot.ChatGroup:
		err := client.database.SetUsernameID(messenger, message.Chat.Title, chatID)
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(message.Text, "/start") {
			return fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", message.Chat.Title), nil
		}

		return "", nil
	}
	return "I don't understand you :(", nil
}
