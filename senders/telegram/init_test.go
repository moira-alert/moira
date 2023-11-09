package telegram

import (
	"fmt"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const telegramType = "telegram"

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{}
		Convey("Empty map", func() {
			sendersNameToType := make(map[string]string)
			err := sender.Init(map[string]interface{}{}, logger, nil, "", sendersNameToType)
			So(err, ShouldResemble, fmt.Errorf("can not read telegram api_token from config"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Has settings", func() {
			sendersNameToType := make(map[string]string)
			senderSettings := map[string]interface{}{
				"type":      telegramType,
				"api_token": "123",
				"front_uri": "http://moira.uri",
			}

			_ = sender.Init(senderSettings, logger, location, "15:04", sendersNameToType)
			So(sender.apiToken, ShouldResemble, "123")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
			So(sendersNameToType[telegramType], ShouldEqual, senderSettings["type"])
		})
	})
}
