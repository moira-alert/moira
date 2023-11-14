package mattermost_test

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/mattermost"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const mattermostType = "mattermost"

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)

	Convey("Init tests", t, func() {
		sender := &mattermost.Sender{}

		Convey("No url", func() {
			senderSettings := map[string]interface{}{
				"type":         mattermostType,
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       nil,
			}

			err := sender.Init(opts)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty url", func() {
			senderSettings := map[string]interface{}{
				"type":         mattermostType,
				"url":          "",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       nil,
			}

			err := sender.Init(opts)
			So(err, ShouldNotBeNil)
		})

		Convey("No api_token", func() {
			senderSettings := map[string]interface{}{
				"type":      mattermostType,
				"url":       "qwerty",
				"front_uri": "qwerty",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       nil,
			}

			err := sender.Init(opts)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty api_token", func() {
			senderSettings := map[string]interface{}{
				"type":      mattermostType,
				"url":       "qwerty",
				"front_uri": "qwerty",
				"api_token": "",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       nil,
			}

			err := sender.Init(opts)
			So(err, ShouldNotBeNil)
		})

		Convey("No front_uri", func() {
			senderSettings := map[string]interface{}{
				"type":      mattermostType,
				"url":       "qwerty",
				"api_token": "qwerty",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       nil,
			}

			err := sender.Init(opts)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty front_uri", func() {
			senderSettings := map[string]interface{}{
				"type":      mattermostType,
				"url":       "qwerty",
				"api_token": "qwerty",
				"front_uri": "",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       nil,
			}

			err := sender.Init(opts)
			So(err, ShouldNotBeNil)
		})

		Convey("Full config", func() {
			senderSettings := map[string]interface{}{
				"type":         mattermostType,
				"url":          "qwerty",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       nil,
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)
		})
	})
}
