package mattermost_test

import (
	"testing"

	"github.com/moira-alert/moira/senders/mattermost"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	Convey("Init tests", t, func() {
		sender := &mattermost.Sender{}

		Convey("No url", func() {
			senderSettings := map[string]interface{}{
				"api_token":    "qwerty",
				"front_url":    "qwerty",
				"insecure_tls": "true",
			}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty url", func() {
			senderSettings := map[string]interface{}{
				"url":          "",
				"api_token":    "qwerty",
				"front_url":    "qwerty",
				"insecure_tls": "true",
			}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("No api_token", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "front_url": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty api_token", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "front_url": "qwerty", "api_token": ""}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("No front_url", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "api_token": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty front_url", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "api_token": "qwerty", "front_url": ""}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Full config", func() {
			senderSettings := map[string]interface{}{
				"url":          "qwerty",
				"api_token":    "qwerty",
				"front_url":    "qwerty",
				"insecure_tls": "true",
			}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldBeNil)
		})
	})
}
