package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/tucnak/telebot.v2"
)

// handleMessage handles incoming messages to start sending events to subscribers chats
func (sender *Sender) handleMessage(message *telebot.Message) error {
	var err error
	id := strconv.FormatInt(message.Chat.ID, 10)
	title := message.Chat.Title
	userTitle := strings.Trim(fmt.Sprintf("%s %s", message.Sender.FirstName, message.Sender.LastName), " ")
	username := message.Chat.Username
	chatType := message.Chat.Type
	switch {
	case chatType == "private" && message.Text == "/start":
		if username == "" {
			sender.bot.Send(message.Chat, "Username is empty. Please add username in Telegram.")
		} else {
			err = sender.DataBase.SetUsernameID(messenger, "@"+username, id)
			if err != nil {
				return err
			}
			sender.bot.Send(message.Chat, fmt.Sprintf("Okay, %s, your id is %s", userTitle, id))
		}
	case chatType == "supergroup" || chatType == "group":
		uid, _ := sender.DataBase.GetIDByUsername(messenger, title)
		if uid == "" {
			sender.bot.Send(message.Chat, fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", title))
		}
		fmt.Println(chatType, title)
		err = sender.DataBase.SetUsernameID(messenger, title, id)
		if err != nil {
			return err
		}
	default:
		sender.bot.Send(message.Chat, "I don't understand you :(")
	}
	return err
}
