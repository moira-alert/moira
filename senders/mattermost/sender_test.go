package mattermost

import (
	"testing"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	Convey("Init tests", t, func() {
		sender := &Sender{}

		Convey("No url", func() {
			senderSettings := map[string]interface{}{
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty url", func() {
			senderSettings := map[string]interface{}{
				"url":          "",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("No api_token", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "front_uri": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty api_token", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "front_uri": "qwerty", "api_token": ""}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("No front_uri", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "api_token": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty front_uri", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "api_token": "qwerty", "front_uri": ""}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Full config", func() {
			senderSettings := map[string]interface{}{
				"url":          "qwerty",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldBeNil)
		})
	})
}
