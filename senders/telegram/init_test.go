package telegram

import (
	"fmt"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{}
		Convey("Empty map", func() {
			err := sender.Init(map[string]string{}, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("can not read telegram api_token from config"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]string{
				"api_token": "123",
				"front_uri": "http://moira.uri",
			}
			_ = sender.Init(senderSettings, logger, location, "15:04")
			So(sender.apiToken, ShouldResemble, "123")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
		})
	})
}

func TestSender_getPollerTimeout(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "error", "test", true)
	Convey("Check getPollerTimeout", t, func() {
		sender := Sender{logger: logger}

		Convey("Not set timeout, should use default value", func() {
			timeout := sender.getPollerTimeout("")
			So(timeout, ShouldResemble, defaultPollerTimeout)
		})

		Convey("Error set, should use default value", func() {
			timeout := sender.getPollerTimeout(";zfrk")
			So(timeout, ShouldResemble, defaultPollerTimeout)
		})

		Convey("Successfully set timeout", func() {
			timeout := sender.getPollerTimeout("60")
			So(timeout, ShouldResemble, time.Minute)
		})
	})
}
