package telegram

import (
	"fmt"
	"testing"

	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_telegram "github.com/moira-alert/moira/mock/notifier/telegram"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
	"gopkg.in/telebot.v3"
)

func TestGetResponseMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	bot := mock_telegram.NewMockBot(mockCtrl)

	Convey("Test get response message", t, func() {
		sender := Sender{DataBase: dataBase, bot: bot}
		Convey("Private chat and bad message", func() {
			message := &telebot.Message{
				Chat: &telebot.Chat{
					Type: telebot.ChatPrivate,
				},
				Text: "/Start",
			}
			response, err := sender.getResponseMessage(message)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, "I don't understand you :(")
		})

		Convey("Private channel", func() {
			message := &telebot.Message{
				Chat: &telebot.Chat{
					Type: telebot.ChatChannelPrivate,
				},
				Text: "/Start",
			}
			response, err := sender.getResponseMessage(message)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, "I don't understand you :(")
		})

		Convey("Private chat and /start command", func() {
			message := &telebot.Message{
				Chat: &telebot.Chat{
					ID:   123,
					Type: telebot.ChatPrivate,
				},
				Text: "/start",
				Sender: &telebot.User{
					FirstName: "FirstName",
					LastName:  "LastName",
				},
			}
			Convey("no username", func() {
				response, err := sender.getResponseMessage(message)
				So(err, ShouldBeNil)
				So(response, ShouldResemble, "Username is empty. Please add username in Telegram.")
			})

			Convey("has username", func() {
				message.Chat.Username = "User"
				Convey("error while save username", func() {
					dataBase.EXPECT().SetUsernameChat(messenger, "@User", `{"chat_id":123}`).Return(fmt.Errorf("error =("))
					response, err := sender.getResponseMessage(message)
					So(err.Error(), ShouldResemble, "failed to set username chat: error =(")
					So(response, ShouldBeEmpty)
				})

				Convey("success send", func() {
					dataBase.EXPECT().SetUsernameChat(messenger, "@User", `{"chat_id":123}`).Return(nil)
					response, err := sender.getResponseMessage(message)
					So(err, ShouldBeNil)
					So(response, ShouldResemble, "Okay, FirstName LastName, your id is 123")
				})
			})
		})

		Convey("Group and SuperGroup", func() {
			groupMessage := &telebot.Message{
				Chat: &telebot.Chat{
					ID:    123,
					Type:  telebot.ChatGroup,
					Title: "MyGroup",
				},
				Text: "/start@MoiraBot",
			}
			superGroupMessage := &telebot.Message{
				Chat: &telebot.Chat{
					ID:    124,
					Type:  telebot.ChatSuperGroup,
					Title: "MySuperGroup",
				},
				Text: "/start@MoiraBot",
			}
			messages := []*telebot.Message{groupMessage, superGroupMessage}

			Convey("SetUsernameChat returns error", func() {
				for _, message := range messages {
					dataBase.EXPECT().SetUsernameChat(messenger, message.Chat.Title, fmt.Sprintf(`{"chat_id":%d}`, message.Chat.ID)).Return(fmt.Errorf("error"))
					response, err := sender.getResponseMessage(message)
					So(err.Error(), ShouldResemble, "failed to set username chat: error")
					So(response, ShouldBeEmpty)
				}
			})

			Convey("SetUsernameChat returns empty error", func() {
				for _, message := range messages {
					dataBase.EXPECT().SetUsernameChat(messenger, message.Chat.Title, fmt.Sprintf(`{"chat_id":%d}`, message.Chat.ID)).Return(nil)
					response, err := sender.getResponseMessage(message)
					So(err, ShouldBeNil)
					So(response, ShouldResemble, fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", message.Chat.Title))
				}
			})

			Convey("Wrong text", func() {
				wrongGroupMessage := &telebot.Message{
					Chat: &telebot.Chat{
						ID:    123,
						Type:  telebot.ChatGroup,
						Title: "MyGroup",
					},
					Text: "please don't answer me",
				}
				wrongSuperGroupMessage := &telebot.Message{
					Chat: &telebot.Chat{
						ID:    124,
						Type:  telebot.ChatSuperGroup,
						Title: "MySuperGroup",
					},
					Text: "please don't answer me",
				}
				wrongMessages := []*telebot.Message{wrongGroupMessage, wrongSuperGroupMessage}
				for _, message := range wrongMessages {
					dataBase.EXPECT().SetUsernameChat(messenger, message.Chat.Title, fmt.Sprintf(`{"chat_id":%d}`, message.Chat.ID)).Return(nil)
					response, err := sender.getResponseMessage(message)
					So(err, ShouldBeNil)
					So(response, ShouldResemble, "")
				}
			})
		})
	})
}
