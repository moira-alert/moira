package slack

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/nlopes/slack"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	Convey("Init tests", t, func(c C) {
		sender := Sender{}
		senderSettings := map[string]string{}
		Convey("Empty map", t, func(c C) {
			err := sender.Init(senderSettings, logger, nil, "")
			c.So(err, ShouldResemble, fmt.Errorf("can not read slack api_token from config"))
			c.So(sender, ShouldResemble, Sender{})
		})

		Convey("has api_token", t, func(c C) {
			senderSettings["api_token"] = "123"
			client := slack.New("123")

			Convey("use_emoji not set", t, func(c C) {
				err := sender.Init(senderSettings, logger, nil, "")
				c.So(err, ShouldBeNil)
				c.So(sender, ShouldResemble, Sender{logger: logger, client: client})
			})

			Convey("use_emoji set to false", t, func(c C) {
				senderSettings["use_emoji"] = "false"
				err := sender.Init(senderSettings, logger, nil, "")
				c.So(err, ShouldBeNil)
				c.So(sender, ShouldResemble, Sender{logger: logger, client: client})
			})

			Convey("use_emoji set to true", t, func(c C) {
				senderSettings["use_emoji"] = "true"
				err := sender.Init(senderSettings, logger, nil, "")
				c.So(err, ShouldBeNil)
				c.So(sender, ShouldResemble, Sender{logger: logger, useEmoji: true, client: client})
			})

			Convey("use_emoji set to something wrong", t, func(c C) {
				senderSettings["use_emoji"] = "123"
				err := sender.Init(senderSettings, logger, nil, "")
				c.So(err, ShouldBeNil)
				c.So(sender, ShouldResemble, Sender{logger: logger, useEmoji: false, client: client})
			})
		})
	})
}

func TestUseDirectMessaging(t *testing.T) {
	Convey("TestUseDirectMessaging", t, func(c C) {
		c.So(useDirectMessaging(""), ShouldBeFalse)
		c.So(useDirectMessaging("contact"), ShouldBeFalse)
		c.So(useDirectMessaging("@contact"), ShouldBeTrue)
		c.So(useDirectMessaging("#contact"), ShouldBeFalse)
	})
}

func TestGetStateEmoji(t *testing.T) {
	sender := Sender{}
	Convey("Use emoji is false", t, func(c C) {
		c.So(sender.getStateEmoji(moira.StateERROR), ShouldResemble, "")
	})

	Convey("Use emoji is true", t, func(c C) {
		sender := Sender{useEmoji: true}
		c.So(sender.getStateEmoji(moira.StateOK), ShouldResemble, okEmoji)
		c.So(sender.getStateEmoji(moira.StateWARN), ShouldResemble, warnEmoji)
		c.So(sender.getStateEmoji(moira.StateERROR), ShouldResemble, errorEmoji)
		c.So(sender.getStateEmoji(moira.StateNODATA), ShouldResemble, nodataEmoji)
		c.So(sender.getStateEmoji(moira.StateEXCEPTION), ShouldResemble, exceptionEmoji)
		c.So(sender.getStateEmoji(moira.StateTEST), ShouldResemble, testEmoji)
	})
}

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	value := float64(123)
	message := "This is message"

	Convey("Build Moira Message tests", t, func(c C) {
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
		}

		Convey("Print moira message with one event", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n```\n02:40: Metric = 123 (OK to NODATA)```"
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "*NODATA*\n```\n02:40: Metric = 123 (OK to NODATA)```"
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", t, func(c C) {
			event.Message = &message
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n```\n02:40: Metric = 123 (OK to NODATA). This is message```"
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n```\n02:40: Metric = 123 (OK to NODATA)```\nPlease, *fix your system or tune this trigger* to generate less events."
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := "*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n```\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)```"
			c.So(actual, ShouldResemble, expected)
		})

		events := make([]moira.NotificationEvent, 0)
		Convey("Print moira message with 1129 events and cutoff", t, func(c C) {
			for i := 0; i < 1200; i++ {
				events = append(events, event)
			}
			lines := strings.Repeat("\n02:40: Metric = 123 (OK to NODATA)", 1129)
			actual := sender.buildMessage(events, trigger, false)
			expected := fmt.Sprintf("*NODATA* [tag1][tag2] <http://moira.url/trigger/TriggerID|Name>\n```%s```\n\n...and 71 more events.", lines)
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger name", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			expected := "*NODATA* Name\n```\n02:40: Metric = 123 (OK to NODATA)```"
			c.So(actual, ShouldResemble, expected)
		})

	})
}
