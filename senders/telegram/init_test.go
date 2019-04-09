package telegram

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func(c C) {
		sender := Sender{}
		Convey("Empty map", t, func(c C) {
			err := sender.Init(map[string]string{}, logger, nil, "")
			c.So(err, ShouldResemble, fmt.Errorf("can not read telegram api_token from config"))
			c.So(sender, ShouldResemble, Sender{})
		})

		Convey("Has settings", t, func(c C) {
			senderSettings := map[string]string{
				"api_token": "123",
				"front_uri": "http://moira.uri",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			c.So(sender.apiToken, ShouldResemble, "123")
			c.So(sender.frontURI, ShouldResemble, "http://moira.uri")
			c.So(sender.logger, ShouldResemble, logger)
			c.So(sender.location, ShouldResemble, location)
		})
	})
}
