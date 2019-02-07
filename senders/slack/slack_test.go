package slack

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/nlopes/slack"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
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
				So(sender, ShouldResemble, Sender{logger: logger, client: client})
			})

			Convey("use_emoji set to false", func() {
				senderSettings["use_emoji"] = "false"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{logger: logger, client: client})
			})

			Convey("use_emoji set to true", func() {
				senderSettings["use_emoji"] = "true"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{logger: logger, useEmoji: true, client: client})
			})

			Convey("use_emoji set to something wrong", func() {
				senderSettings["use_emoji"] = "123"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{logger: logger, useEmoji: false, client: client})
			})
		})
	})
}
