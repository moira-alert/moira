package slack

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira/senders/message_builder"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/slack-go/slack"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	Convey("Init tests", t, func() {
		sender := Sender{}
		senderSettings := map[string]string{}
		Convey("Empty map", func() {
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("can not read slack api_token from config"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("has api_token", func() {
			senderSettings["api_token"] = "123"
			client := slack.New("123")

			Convey("use_emoji not set", func() {
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{MessageBuilder: message_builder.NewMessageBuilder("", messageMaxCharacters, logger, nil), client: client})
			})

			Convey("use_emoji set to false", func() {
				senderSettings["use_emoji"] = "false"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{MessageBuilder: message_builder.NewMessageBuilder("", messageMaxCharacters, logger, nil), client: client})
			})

			Convey("use_emoji set to true", func() {
				senderSettings["use_emoji"] = "true"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{useEmoji: true, MessageBuilder: message_builder.NewMessageBuilder("", messageMaxCharacters, logger, nil), client: client})
			})

			Convey("use_emoji set to something wrong", func() {
				senderSettings["use_emoji"] = "123"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{MessageBuilder: message_builder.NewMessageBuilder("", messageMaxCharacters, logger, nil), client: client})
			})
		})
	})
}

func TestUseDirectMessaging(t *testing.T) {
	Convey("TestUseDirectMessaging", t, func() {
		So(useDirectMessaging(""), ShouldBeFalse)
		So(useDirectMessaging("contact"), ShouldBeFalse)
		So(useDirectMessaging("@contact"), ShouldBeTrue)
		So(useDirectMessaging("#contact"), ShouldBeFalse)
	})
}

func TestGetStateEmoji(t *testing.T) {
	sender := Sender{}
	Convey("Use emoji is false", t, func() {
		So(sender.getStateEmoji(moira.StateERROR), ShouldResemble, "")
	})

	Convey("Use emoji is true", t, func() {
		sender := Sender{useEmoji: true}
		So(sender.getStateEmoji(moira.StateOK), ShouldResemble, okEmoji)
		So(sender.getStateEmoji(moira.StateWARN), ShouldResemble, warnEmoji)
		So(sender.getStateEmoji(moira.StateERROR), ShouldResemble, errorEmoji)
		So(sender.getStateEmoji(moira.StateNODATA), ShouldResemble, nodataEmoji)
		So(sender.getStateEmoji(moira.StateEXCEPTION), ShouldResemble, exceptionEmoji)
		So(sender.getStateEmoji(moira.StateTEST), ShouldResemble, testEmoji)
	})
}
