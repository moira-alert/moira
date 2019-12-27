package telegram

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/tucnak/telebot.v2"
)

func TestGetResponseMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Test get response message", t, func() {
		sender := Sender{DataBase: dataBase}
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
					dataBase.EXPECT().SetUsernameID(messenger, "@User", "123").Return(fmt.Errorf("error =("))
					response, err := sender.getResponseMessage(message)
					So(err, ShouldResemble, fmt.Errorf("error =("))
					So(response, ShouldBeEmpty)
				})

				Convey("success send", func() {
					dataBase.EXPECT().SetUsernameID(messenger, "@User", "123").Return(nil)
					response, err := sender.getResponseMessage(message)
					So(err, ShouldBeNil)
					So(response, ShouldResemble, "Okay, FirstName LastName, your id is 123")
				})
			})
		})

		Convey("SuperGroup", func() {
			message := &telebot.Message{
				Chat: &telebot.Chat{
					ID:    123,
					Type:  telebot.ChatSuperGroup,
					Title: "MyGroup",
				},
			}

			Convey("GetIDByUsername returns error", func() {
				dataBase.EXPECT().GetIDByUsername(messenger, message.Chat.Title).Return("", fmt.Errorf("error"))
				dataBase.EXPECT().SetUsernameID(messenger, message.Chat.Title, "123").Return(nil)
				response, err := sender.getResponseMessage(message)
				So(err, ShouldBeNil)
				So(response, ShouldResemble, fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", message.Chat.Title))
			})

			Convey("GetIDByUsername returns empty uuid", func() {
				dataBase.EXPECT().GetIDByUsername(messenger, message.Chat.Title).Return("", nil)

				Convey("SetUsernameID returns error", func() {
					dataBase.EXPECT().SetUsernameID(messenger, message.Chat.Title, "123").Return(fmt.Errorf("error"))
					response, err := sender.getResponseMessage(message)
					So(err, ShouldResemble, fmt.Errorf("error"))
					So(response, ShouldBeEmpty)
				})

				Convey("SetUsernameID returns empty error", func() {
					dataBase.EXPECT().SetUsernameID(messenger, message.Chat.Title, "123").Return(nil)
					response, err := sender.getResponseMessage(message)
					So(err, ShouldBeNil)
					So(response, ShouldResemble, fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", message.Chat.Title))
				})
			})

			Convey("GetIDByUsername return uuid", func() {
				dataBase.EXPECT().GetIDByUsername(messenger, message.Chat.Title).Return("123", nil)

				Convey("SetUsernameID returns error", func() {
					dataBase.EXPECT().SetUsernameID(messenger, message.Chat.Title, "123").Return(fmt.Errorf("error"))
					response, err := sender.getResponseMessage(message)
					So(err, ShouldResemble, fmt.Errorf("error"))
					So(response, ShouldBeEmpty)
				})

				Convey("SetUsernameID returns empty error", func() {
					dataBase.EXPECT().SetUsernameID(messenger, message.Chat.Title, "123").Return(nil)
					response, err := sender.getResponseMessage(message)
					So(err, ShouldBeNil)
					So(response, ShouldBeEmpty)
				})
			})
		})
	})
}
