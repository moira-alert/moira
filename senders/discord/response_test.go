package discord

import (
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetResponseMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	message := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author: &discordgo.User{},
		},
	}
	channel := &discordgo.Channel{}

	Convey("getResponse tests", t, func() {
		sender := Sender{DataBase: dataBase, botUserID: "123"}

		Convey("Message from self", func() {
			message.Author.ID = sender.botUserID
			response, err := sender.getResponse(message, channel)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, "")
		})

		Convey("Message is not !start", func() {
			message.Author.ID = "456"
			message.Content = "not !start"
			response, err := sender.getResponse(message, channel)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, "")
		})

		Convey("DM channel", func() {
			dataBase.EXPECT().SetUsernameID(messenger, "@User", "456").Return(nil)
			channel.Type = discordgo.ChannelTypeDM
			channel.ID = "456"
			message.Content = "!start"
			message.Author.Username = "User"
			response, err := sender.getResponse(message, channel)
			So(err, ShouldBeNil)
			msg := fmt.Sprintf("Okay, %s, your id is %s", message.Author.Username, channel.ID)
			So(response, ShouldResemble, msg)
		})

		Convey("Guild Text channel", func() {
			dataBase.EXPECT().SetUsernameID(messenger, "testchan", "456").Return(nil)
			channel.Type = discordgo.ChannelTypeGuildText
			channel.ID = "456"
			channel.Name = "testchan"
			message.Content = "!start"
			response, err := sender.getResponse(message, channel)
			So(err, ShouldBeNil)
			msg := fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", channel.Name)
			So(response, ShouldResemble, msg)
		})

		Convey("unsupported channel", func() {
			channel.Type = discordgo.ChannelTypeGuildVoice
			response, err := sender.getResponse(message, channel)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, "Unsupported channel type")
		})
	})
}
