//go:build manual

package mattermost_test

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/mattermost"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

// TestSender is integration manual test. Paste your Url, Token and Channel ID and check message in Mattermost.
func TestSender(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)

	const (
		url       = "http://localhost:8065"
		apiToken  = "8pdo6yoiutgidgxs9qxhbo7w4h"
		channelID = "3y6ab8rptfdr9m1hzskghpxwsc"
	)

	Convey("Init tests", t, func() {
		sender := &mattermost.Sender{}

		Convey("With url and apiToken", func() {
			senderSettings := map[string]string{
				"url":          url,
				"api_token":    apiToken,
				"front_uri":    "http://moira.url",
				"insecure_tls": "true",
			}
			location, _ := time.LoadLocation("UTC")
			err := sender.Init(senderSettings, logger, location, "")
			So(err, ShouldBeNil)

			event := moira.NotificationEvent{
				TriggerID: "TriggerID",
				Values:    map[string]float64{"t1": 123},
				Timestamp: 150000000,
				Metric:    "Metric",
				OldState:  moira.StateOK,
				State:     moira.StateNODATA,
			}
			events, contact, trigger, plots, throttled := moira.NotificationEvents{event}, moira.Contact{
				Value: channelID,
			}, moira.TriggerData{
				ID:   "ID",
				Name: "Name",
			}, make([][]byte, 0), false

			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
		})
	})
}
