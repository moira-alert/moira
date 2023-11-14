package mattermost_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/senders/mattermost"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Init tests", t, func() {
		sender := &mattermost.Sender{}

		Convey("No url", func() {
			senderSettings := map[string]interface{}{
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}
			err := sender.Init(senderSettings, logger, nil, "", dataBase)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty url", func() {
			senderSettings := map[string]interface{}{
				"url":          "",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}
			err := sender.Init(senderSettings, logger, nil, "", dataBase)
			So(err, ShouldNotBeNil)
		})

		Convey("No api_token", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "front_uri": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "", dataBase)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty api_token", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "front_uri": "qwerty", "api_token": ""}
			err := sender.Init(senderSettings, logger, nil, "", dataBase)
			So(err, ShouldNotBeNil)
		})

		Convey("No front_uri", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "api_token": "qwerty"}
			err := sender.Init(senderSettings, logger, nil, "", dataBase)
			So(err, ShouldNotBeNil)
		})

		Convey("Empty front_uri", func() {
			senderSettings := map[string]interface{}{"url": "qwerty", "api_token": "qwerty", "front_uri": ""}
			err := sender.Init(senderSettings, logger, nil, "", dataBase)
			So(err, ShouldNotBeNil)
		})

		Convey("Full config", func() {
			senderSettings := map[string]interface{}{
				"url":          "qwerty",
				"api_token":    "qwerty",
				"front_uri":    "qwerty",
				"insecure_tls": true,
			}
			err := sender.Init(senderSettings, logger, nil, "", dataBase)
			So(err, ShouldBeNil)
		})
	})
}
