package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/telebot.v3"
)

// handleMessage handles incoming messages to start sending events to subscribers chats.
func (sender *Sender) handleMessage(message *telebot.Message) error {
	responseMessage, err := sender.getResponseMessage(message)
	if err != nil {
		return err
	}
	if responseMessage != "" {
		if _, err = sender.bot.Reply(message, responseMessage); err != nil {
			return sender.removeTokenFromError(err)
		}
	}
	return nil
}

func (sender *Sender) getResponseMessage(message *telebot.Message) (string, error) {
	chatID := strconv.FormatInt(message.Chat.ID, 10)
	switch {
	case message.Chat.Type == telebot.ChatPrivate && message.Text == "/start":
		if message.Chat.Username == "" {
			return "Username is empty. Please add username in Telegram.", nil
		}
		_, err := sender.setChat(message)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Okay, %s, your id is %s", strings.Trim(fmt.Sprintf("%s %s", message.Sender.FirstName, message.Sender.LastName), " "), chatID), nil
	case (message.Chat.Type == telebot.ChatSuperGroup || message.Chat.Type == telebot.ChatGroup):
		contactValue, err := sender.getContactValueByMessage(message)
		if err != nil {
			return "", fmt.Errorf("failed to get contact value from message: %w", err)
		}

		_, err = sender.setChat(message)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(message.Text, "/start") {
			return fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", contactValue), nil
		}
		return "", nil
	}
	return "I don't understand you :(", nil
}
