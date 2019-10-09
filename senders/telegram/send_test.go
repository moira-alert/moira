package telegram

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 97.4458331200185},
			Timestamp: 150000000,
			Metric:    "Metric name",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Trigger Name",
			ID:   "TriggerID",
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID
`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger Name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false, messageMaxCharacters)
			expected := `💣NODATA Name  (1)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false, messageMaxCharacters)
			expected := `💣NODATA   (1)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			event.TriggerID = ""
			trigger.ID = ""
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})

		events := make([]moira.NotificationEvent, 0)
		Convey("Print moira message with 6 events and photo message length", func() {
			for i := 0; i < 18; i++ {
				events = append(events, event)
			}
			actual := sender.buildMessage(events, trigger, false, albumCaptionMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (18)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)

...and 8 more events.

http://moira.url/trigger/TriggerID
`
			fmt.Printf("Bytes: %v\n", len(expected))
			fmt.Printf("Symbols: %v\n", len([]rune(expected)))
			So(actual, ShouldResemble, expected)
		})
	})
}
