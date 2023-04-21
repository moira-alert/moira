package default_sender

import (
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira"

	. "github.com/smartystreets/goconvey/convey"
)

const messageMaxCharacters = 500

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := DefaultSender{
		location:             location,
		frontURI:             "http://moira.url",
		messageMaxCharacters: messageMaxCharacters,
	}

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

	Convey("Build Moira Message tests", t, func() {
		Convey("Print moira message with one event", func() {
			actual := sender.BuildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.BuildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "*NODATA*\n```\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.BuildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40: Metric = 123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			event.MessageEventInfo = nil
			actual := sender.BuildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40: Metric = 123 (OK to NODATA)```\nPlease, *fix your system or tune this trigger* to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.BuildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := "*NODATA* <http://moira.url/trigger/TriggerID|Name> [tag1][tag2]\n" + slackCompatibleMD +
				"\n\n```\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger name", func() {
			actual := sender.BuildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			expected := "*NODATA* Name\n```\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with desc + events < msgLimit", func() {
			events, eventsString := makeEvents(event, messageMaxCharacters/2-200)
			desc := makeDesc(messageMaxCharacters/2 + 100)
			actual := sender.BuildMessage(events, moira.TriggerData{Desc: desc}, false)
			expected := "*NODATA*\n" + desc + "\n```" + eventsString + "```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message desc more than can be", func() {
			events, _ := makeEvents(event, messageMaxCharacters/2)
			desc := makeDesc(messageMaxCharacters)
			actual := sender.BuildMessage(events, moira.TriggerData{Desc: desc}, false)
			expected := "*NODATA*\n" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\n```\n" +
				"02:40: Metric = 123 (OK to NODATA)\n" +
				"02:40: Metric = 123 (OK to NODATA)\n" +
				"02:40: Metric = 123 (OK to NODATA)```" +
				"\n...and 4 more events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Events less than maximum available", func() {
			events, _ := makeEvents(event, messageMaxCharacters/5)
			desc := makeDesc(messageMaxCharacters)
			actual := sender.BuildMessage(events, moira.TriggerData{Desc: desc}, false)
			expected := "*NODATA*\n" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\n```\n" +
				"02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildDescription(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := DefaultSender{location: location, frontURI: "http://moira.url"}
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

func makeEvents(event moira.NotificationEvent, count int) (moira.NotificationEvents, string) {
	eventLine := "\n02:40: Metric = 123 (OK to NODATA)"
	oneEventLineLen := len([]rune(eventLine))
	// Events list with chars less than half the message limit
	var (
		events       moira.NotificationEvents
		eventsString string
	)
	for i := 0; i < count/oneEventLineLen; i++ {
		events = append(events, event)
		eventsString += eventLine
	}

	return events, eventsString
}

func makeDesc(count int) string {
	return strings.Repeat("a", count)
}
