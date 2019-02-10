package slack

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/nlopes/slack"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	Convey("Init tests", t, func() {
		sender := Sender{}
		senderSettings := map[string]string{}
		Convey("Empty map", func() {
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("can not read slack api_token from config"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("has api_token", func() {
			senderSettings["api_token"] = "123"
			client := slack.New("123")

			Convey("use_emoji not set", func() {
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{logger: logger, client: client})
			})

			Convey("use_emoji set to false", func() {
				senderSettings["use_emoji"] = "false"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{logger: logger, client: client})
			})

			Convey("use_emoji set to true", func() {
				senderSettings["use_emoji"] = "true"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{logger: logger, useEmoji: true, client: client})
			})

			Convey("use_emoji set to something wrong", func() {
				senderSettings["use_emoji"] = "123"
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldBeNil)
				So(sender, ShouldResemble, Sender{logger: logger, useEmoji: false, client: client})
			})
		})
	})
}

func TestUseDirectMessaging(t *testing.T) {
	Convey("TestUseDirectMessaging", t, func() {
		So(useDirectMessaging(""), ShouldBeFalse)
		So(useDirectMessaging("contact"), ShouldBeFalse)
		So(useDirectMessaging("@contact"), ShouldBeTrue)
		So(useDirectMessaging("#contact"), ShouldBeFalse)
	})
}

func TestGetStateEmoji(t *testing.T) {
	sender := Sender{}
	Convey("Use emoji is false", t, func() {
		So(sender.getStateEmoji("ERROR"), ShouldResemble, "")
	})

	Convey("Use emoji is true", t, func() {
		sender := Sender{useEmoji: true}
		So(sender.getStateEmoji("OK"), ShouldResemble, okEmoji)
		So(sender.getStateEmoji("WARN"), ShouldResemble, warnEmoji)
		So(sender.getStateEmoji("ERROR"), ShouldResemble, errorEmoji)
		So(sender.getStateEmoji("NODATA"), ShouldResemble, nodataEmoji)
		So(sender.getStateEmoji("EXCEPTION"), ShouldResemble, exceptionEmoji)
		So(sender.getStateEmoji("TEST"), ShouldResemble, testEmoji)
	})
}

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
			OldState:  "OK",
			State:     "NODATA",
			Message:   nil,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n  \n```\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "*NODATA*  <http://moira.url/trigger/TriggerID|>\n  \n```\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			event.Message = &message
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n  \n```\n02:40: Metric = 123 (OK to NODATA). This is message```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n  \n```\n02:40: Metric = 123 (OK to NODATA)```\nPlease, *fix your system or tune this trigger* to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n  \n```\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildMessageTriggerIDEmpty(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	value := float64(123)
	message := "This is message"

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  "OK",
			State:     "NODATA",
			Message:   nil,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "",
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] Name\n  \n```\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "*NODATA*  \n  \n```\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			event.Message = &message
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] Name\n  \n```\n02:40: Metric = 123 (OK to NODATA). This is message```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := "*NODATA* [tag1][tag2] Name\n  \n```\n02:40: Metric = 123 (OK to NODATA)```\nPlease, *fix your system or tune this trigger* to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] Name\n  \n```\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})
	})
}
