package slack

import (
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	Convey("Init tests", t, func() {
		sender := Sender{}
		senderSettings := map[string]interface{}{}
		Convey("Empty map", func() {
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("has empty api_token", func() {
			senderSettings["api_token"] = ""
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldNotBeNil)
		})

		Convey("has api_token", func() {
			senderSettings["api_token"] = "123"

			Convey("use_emoji set to something wrong", func() {
				senderSettings["use_emoji"] = 123
				err := sender.Init(senderSettings, logger, nil, "")
				So(err, ShouldNotBeNil)
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

func TestBuildMessage(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")

	Convey("Build Moira Message tests", t, func() {
		sender := &Sender{}
		senderSettings := map[string]interface{}{
			"use_emoji": false,
			"front_uri": "http://moira.url",
			"api_token": "qwerty",
		}
		initErr := sender.Init(senderSettings, logger, location, moira.DefaultDateTimeFormat)
		So(initErr, ShouldBeNil)

		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
			Desc: `# header1
some text **bold text**
## header 2
some other text _italic text_`,
		}

		slackCompatibleMD := `*header1*
some text *bold text*

*header 2*
some other text italic text
`

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "*NODATA*\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.\n```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```\nPlease, *fix your system or tune this trigger* to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			expected := "*NODATA* Name\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(actual, ShouldResemble, expected)
		})

		eventLine := "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
		oneEventLineLen := len([]rune(eventLine))
		// Events list with chars less than half the message limit
		var shortEvents moira.NotificationEvents
		var shortEventsString string
		for i := 0; i < (messageMaxCharacters/2-200)/oneEventLineLen; i++ {
			shortEvents = append(shortEvents, event)
			shortEventsString += eventLine
		}
		// Events list with chars greater than half the message limit
		var longEvents moira.NotificationEvents
		var longEventsString string
		for i := 0; i < (messageMaxCharacters/2+200)/oneEventLineLen; i++ {
			longEvents = append(longEvents, event)
			longEventsString += eventLine
		}
		longDesc := strings.Repeat("a", messageMaxCharacters/2+100)

		Convey("Print moira message with desc + events < msgLimit", func() {
			actual := sender.buildMessage(shortEvents, moira.TriggerData{Desc: longDesc}, false)
			expected := "*NODATA*\n" + longDesc + "\n```" + shortEventsString + "\n```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message desc > msgLimit/2", func() {
			var events moira.NotificationEvents
			var eventsString string
			for i := 0; i < (messageMaxCharacters/2-10)/oneEventLineLen; i++ {
				events = append(events, event)
				eventsString += eventLine
			}
			actual := sender.buildMessage(events, moira.TriggerData{Desc: longDesc}, false)
			expected := "*NODATA*\naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message events string > msgLimit/2", func() {
			desc := strings.Repeat("a", messageMaxCharacters/2-100)
			actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: desc}, false)
			expected := "*NODATA*\naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```\n...and 3 more events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with both desc and events > msgLimit/2", func() {
			actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: longDesc}, false)
			expected := "*NODATA*\naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\n```\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```\n...and 5 more events."
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildDescription(t *testing.T) {
	Convey("Build desc tests", t, func() {
		trigger := moira.TriggerData{
			Desc: `# header1
some text **bold text**
## header 2
some other text _italic text_`,
		}

		slackCompatibleMD := `*header1*
some text *bold text*

*header 2*
some other text italic text
`

		Convey("Build empty desc", func() {
			actual := descriptionFormatter(moira.TriggerData{Desc: ""})
			expected := ""
			So(actual, ShouldResemble, expected)
		})

		Convey("Build desc with headers and bold", func() {
			actual := descriptionFormatter(trigger)
			expected := slackCompatibleMD + "\n\n"
			So(actual, ShouldResemble, expected)
		})
	})

	Convey("Build desc with lists", t, func() {
		trigger := moira.TriggerData{
			Desc: `
1. a
`,
		}

		expected := ` 1. a


`

		Convey("Expect descriptionFormatter not to panic", func() {
			actual := descriptionFormatter(trigger)

			So(actual, ShouldEqual, expected)
		})
	})
}
