package mattermost_test

import (
	"testing"

	"github.com/moira-alert/moira/notifier/senders/mattermost"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	Convey("Init tests", t, func() {
		Convey("No url", func() {
			senderSettings := map[string]string{
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": "true",
			}
			_, err := mattermost.NewSender(senderSettings, nil)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty url", func() {
			senderSettings := map[string]string{
				"url":          "",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": "true",
			}
			_, err := mattermost.NewSender(senderSettings, nil)
			So(err, ShouldNotBeNil)
		})

		Convey("No api_token", func() {
			senderSettings := map[string]string{"url": "qwerty", "front_uri": "qwerty"}
			_, err := mattermost.NewSender(senderSettings, nil)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty api_token", func() {
			senderSettings := map[string]string{"url": "qwerty", "front_uri": "qwerty", "api_token": ""}
			_, err := mattermost.NewSender(senderSettings, nil)
			So(err, ShouldNotBeNil)
		})

		Convey("No front_uri", func() {
			senderSettings := map[string]string{"url": "qwerty", "api_token": "qwerty"}
			_, err := mattermost.NewSender(senderSettings, nil)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty front_uri", func() {
			senderSettings := map[string]string{"url": "qwerty", "api_token": "qwerty", "front_uri": ""}
			_, err := mattermost.NewSender(senderSettings, nil)
			So(err, ShouldNotBeNil)
		})

		Convey("Full config", func() {
			senderSettings := map[string]string{
				"url":          "qwerty",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": "true",
			}
			_, err := mattermost.NewSender(senderSettings, nil)
			So(err, ShouldBeNil)
		})
	})
}
