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
		So(sender.getStateEmoji(moira.StateERROR), ShouldResemble, "")
	})

	Convey("Use emoji is true", t, func() {
		sender := Sender{useEmoji: true}
		So(sender.getStateEmoji(moira.StateOK), ShouldResemble, okEmoji)
		So(sender.getStateEmoji(moira.StateWARN), ShouldResemble, warnEmoji)
		So(sender.getStateEmoji(moira.StateERROR), ShouldResemble, errorEmoji)
		So(sender.getStateEmoji(moira.StateNODATA), ShouldResemble, nodataEmoji)
		So(sender.getStateEmoji(moira.StateEXCEPTION), ShouldResemble, exceptionEmoji)
		So(sender.getStateEmoji(moira.StateTEST), ShouldResemble, testEmoji)
	})
}

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}

	Convey("Build Moira Message tests", t, func() {
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
				"\n\n```\n02:40: Metric = t1:123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "*NODATA*\n```\n02:40: Metric = t1:123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40: Metric = t1:123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40: Metric = t1:123 (OK to NODATA)```\nPlease, *fix your system or tune this trigger* to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			expected := "*NODATA* Name\n```\n02:40: Metric = t1:123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		eventLine := "\n02:40: Metric = t1:123 (OK to NODATA)"
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
			expected := "*NODATA*\n" + longDesc + "\n```" + shortEventsString + "```"
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
			expected := "*NODATA*\naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\n```\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message events string > msgLimit/2", func() {
			desc := strings.Repeat("a", messageMaxCharacters/2-100)
			actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: desc}, false)
			expected := "*NODATA*\naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n```\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)```\n...and 3 more events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with both desc and events > msgLimit/2", func() {
			actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: longDesc}, false)
			expected := "*NODATA*\naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\n```\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)\n02:40: Metric = t1:123 (OK to NODATA)```\n...and 6 more events."
			So(actual, ShouldResemble, expected)
		})

	})
}

func TestBuildDescription(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
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
			actual := sender.buildDescription(moira.TriggerData{Desc: ""})
			expected := ""
			So(actual, ShouldResemble, expected)
		})

		Convey("Build desc with headers and bold", func() {
			actual := sender.buildDescription(trigger)
			expected := slackCompatibleMD + "\n\n"
			So(actual, ShouldResemble, expected)
		})
	})
}
