package telegram

import (
	"fmt"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewSender(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		Convey("Empty map", func() {
			sender, err := NewSender(map[string]string{}, logger, nil, nil)
			So(err, ShouldResemble, fmt.Errorf("can not read telegram api_token from config"))
			So(sender, ShouldBeNil)
		})

		Convey("Incorrect api token", func() {
			senderSettings := map[string]string{
				"api_token": "123",
				"front_uri": "http://moira.uri",
			}
			sender, err := NewSender(senderSettings, logger, location, nil)
			So(sender, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})
}
