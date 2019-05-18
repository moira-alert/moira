package telegram

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	mdHeaderRegex = regexp.MustCompile(`(?m)^\s*#{1,}\s*(?P<headertext>[^#\n]+)$`)
	value := float64(97.4458331200185)
	message := "This is message"

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric name",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
			Message:   nil,
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

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)
**header1**
some text **bold text**
**header 2**
some other text _italic text_

02:40: Metric name = 97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID
`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger Name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false, messageMaxCharacters)
			expected := `💣NODATA Name  (1)

02:40: Metric name = 97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false, messageMaxCharacters)
			expected := `💣NODATA   (1)

02:40: Metric name = 97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			event.Message = &message
			event.TriggerID = ""
			trigger.ID = ""
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)
**header1**
some text **bold text**
**header 2**
some other text _italic text_

02:40: Metric name = 97.4458331200185 (OK to NODATA). This is message`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)
**header1**
some text **bold text**
**header 2**
some other text _italic text_

02:40: Metric name = 97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})

		events := make([]moira.NotificationEvent, 0)
		Convey("Print moira message with 6 events and photo message length", func() {
			for i := 0; i < 18; i++ {
				events = append(events, event)
			}
			actual := sender.buildMessage(events, trigger, false, photoCaptionMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (18)
**header1**
some text **bold text**
**header 2**
some other text _italic text_

02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)

...and 9 more events.

http://moira.url/trigger/TriggerID
`
			fmt.Println(fmt.Sprintf("Bytes: %v", len(expected)))
			fmt.Println(fmt.Sprintf("Symbols: %v", len([]rune(expected))))
			So(actual, ShouldResemble, expected)
		})
	})
}
