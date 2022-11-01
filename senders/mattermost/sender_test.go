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

		Convey("Empty url", func() {
			senderSettings := map[string]string{"api_token": "qwerty", "front_uri": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty api_token", func() {
			senderSettings := map[string]string{"url": "qwerty", "front_uri": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Empty front_uri", func() {
			senderSettings := map[string]string{"url": "qwerty", "api_token": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("Full config", func() {
			senderSettings := map[string]string{"url": "qwerty", "api_token": "qwerty", "front_uri": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldBeNil)
		})
	})
}
