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

//TestSender is integration test, run it manually with your Url, Token and Channel ID.
//func TestSender(t *testing.T) {
//	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
//
//	Convey("Init tests", t, func() {
//		sender := &mattermost.Sender{}
//
//		Convey("With url and api_token", func() {
//			senderSettings := map[string]string{"url": "http://localhost:8065", "api_token": "8pdo6yoiutgidgxs9qxhbo7w4h", "front_uri": "http://moira.url"}
//			location, _ := time.LoadLocation("UTC")
//			err := sender.Init(senderSettings, logger, location, "")
//			So(err, ShouldBeNil)
//
//			event := moira.NotificationEvent{
//				TriggerID: "TriggerID",
//				Values:    map[string]float64{"t1": 123},
//				Timestamp: 150000000,
//				Metric:    "Metric",
//				OldState:  moira.StateOK,
//				State:     moira.StateNODATA,
//			}
//			events, contact, trigger, plots, throttled := moira.NotificationEvents{event}, moira.ContactData{
//				Value: "3y6ab8rptfdr9m1hzskghpxwsc",
//			}, moira.TriggerData{
//				ID:   "ID",
//				Name: "Name",
//			}, make([][]byte, 0), false
//
//			err = sender.SendEvents(events, contact, trigger, plots, throttled)
//			So(err, ShouldBeNil)
//		})
//	})
//}
