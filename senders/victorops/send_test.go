package victorops

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	value := float64(123)
	message := "This is message"

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
			Message:   nil,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
			Desc: "## test\n **test** `test` test\n",
		}

		strippedDesc := "test\n test test test\n"
		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			event.Message = &message
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA). This is message"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA)\nPlease, fix your system or tune this trigger to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			expected := "\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

	})
}
