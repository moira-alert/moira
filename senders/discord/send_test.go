package discord

import (
	"strings"
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
			Desc: `# header1
some text **bold text**
## header 2
some other text _italic text_`,
		}

		desc := `**header1**
some text **bold text**
**header 2**
some other text _italic text_`

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "NODATA Trigger Name [tag1][tag2] (1)\n" + desc + `

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID
`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger Name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			expected := `NODATA Name  (1)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := `NODATA   (1)

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			event.TriggerID = ""
			trigger.ID = ""
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := "NODATA Trigger Name [tag1][tag2] (1)\n" + desc + `

02:40: Metric name = t1:97.4458331200185 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := "NODATA Trigger Name [tag1][tag2] (1)\n" + desc + `

02:40: Metric name = t1:97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})

		eventLine := "\n02:40: Metric name = t1:97.4458331200185 (OK to NODATA)"
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
			expected := "NODATA   (14)\n" + longDesc + "\n" + shortEventsString
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
			expected := `NODATA   (17)
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...

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
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)`

			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message events string > msgLimit/2", func() {
			desc := strings.Repeat("a", messageMaxCharacters/2-100)
			actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: desc}, false)
			expected := `NODATA   (21)
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

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
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)

...and 3 more events.`

			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with both desc and events > msgLimit/2", func() {
			actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: longDesc}, false)
			expected := `NODATA   (21)
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...

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
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)

...and 4 more events.`

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

		discordCompatibleMD := `**header1**
some text **bold text**
**header 2**
some other text _italic text_
`

		Convey("Build empty desc", func() {
			actual := sender.buildDescription(moira.TriggerData{Desc: ""})
			expected := ""
			So(actual, ShouldResemble, expected)
		})

		Convey("Build desc with headers and bold", func() {
			actual := sender.buildDescription(trigger)
			expected := discordCompatibleMD
			So(actual, ShouldResemble, expected)
		})
	})
}
