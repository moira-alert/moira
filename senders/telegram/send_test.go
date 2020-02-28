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
	value := float64(97.4458331200185)

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric name",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n"+
			"```\n02:40: Metric name = 97.4458331200185 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger Name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false, messageMaxCharacters)
			expected := "*NODATA* Name\n```\n02:40: Metric name = 97.4458331200185 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false, messageMaxCharacters)
			expected := "*NODATA*\n```\n02:40: Metric name = 97.4458331200185 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" +
				"```\n02:40: Metric name = 97.4458331200185 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true, messageMaxCharacters)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" +
				"```\n02:40: Metric name = 97.4458331200185 (OK to NODATA)```\nPlease, *fix your system or tune this trigger* to generate less events."
			So(actual, ShouldResemble, expected)
		})

		events := make([]moira.NotificationEvent, 0)
		Convey("Print moira message with 6 events and photo message length", func() {
			for i := 0; i < 6; i++ {
				events = append(events, event)
			}
			actual := sender.buildMessage(events, trigger, false, photoCaptionMaxCharacters)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" +
				"```\n02:40: Metric name = 97.4458331200185 (OK to NODATA)\n02:40: Metric name = 97.4458331200185 (OK to NODATA)\n02:40: Metric name = 97.4458331200185 (OK to NODATA)\n02:40: Metric name = 97.4458331200185 (OK to NODATA)\n02:40: Metric name = 97.4458331200185 (OK to NODATA)\n02:40: Metric name = 97.4458331200185 (OK to NODATA)```"
			fmt.Println(fmt.Sprintf("Bytes: %v", len(expected)))
			fmt.Println(fmt.Sprintf("Symbols: %v", len([]rune(expected))))
			So(actual, ShouldResemble, expected)
		})
	})
}
