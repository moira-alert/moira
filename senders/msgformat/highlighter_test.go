package msgformat

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/emoji_provider"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	testMaxChars = 4_000
)

func TestFormat(t *testing.T) {
	Convey("Given configured formatter", t, func() {
		location, locationErr := time.LoadLocation("UTC")
		So(locationErr, ShouldBeNil)

		provider, err := emoji_provider.NewEmojiProvider("", nil)
		So(err, ShouldBeNil)
		formatter := NewHighlightSyntaxFormatter(
			provider,
			false,
			"http://moira.url",
			location,
			testUriFormatter,
			testDescriptionFormatter,
			testBoldFormatter,
		)

		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		const shortDesc = `My description`
		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
			Desc: shortDesc,
		}

		Convey("Message with one event", func() {
			events, throttled := moira.NotificationEvents{event}, false
			msg := formatter.Format(getParams(events, trigger, throttled))

			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(msg, ShouldEqual, expected)
		})

		Convey("Message with one event and throttled", func() {
			events, throttled := moira.NotificationEvents{event}, true
			msg := formatter.Format(getParams(events, trigger, throttled))

			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```" + "\n" +
				"Please, *fix your system or tune this trigger* to generate less events."
			So(msg, ShouldEqual, expected)
		})

		Convey("Moira message with 3 events", func() {
			actual := formatter.Format(getParams([]moira.NotificationEvent{event, event, event}, trigger, false))
			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Long message parts", func() {
			const (
				msgLimit        = 4_000
				halfLimit       = msgLimit / 2
				greaterThanHalf = halfLimit + 100
				lessThanHalf    = halfLimit - 100
			)

			const eventLine = "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
			oneEventLineLen := len([]rune(eventLine))

			longDesc := strings.Repeat("a", greaterThanHalf)

			// Events list with chars greater than half of the message limit
			var longEvents moira.NotificationEvents
			for i := 0; i < greaterThanHalf/oneEventLineLen; i++ {
				longEvents = append(longEvents, event)
			}

			Convey("Long description. desc > msgLimit/2", func() {
				var events moira.NotificationEvents
				for i := 0; i < lessThanHalf/oneEventLineLen; i++ {
					events = append(events, event)
				}

				actual := formatter.Format(getParams(events, moira.TriggerData{Desc: longDesc}, false))
				expected := "**NODATA**\n" +
					strings.Repeat("a", 2100) + "\n" +
					"```\n" +
					strings.Repeat("02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n", 39) +
					"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
				So(actual, ShouldResemble, expected)
			})

			Convey("Many events. eventString > msgLimit/2", func() {
				desc := strings.Repeat("a", lessThanHalf)
				actual := formatter.Format(getParams(longEvents, moira.TriggerData{Desc: desc}, false))
				expected := "**NODATA**\n" +
					desc + "\n" +
					"```\n" +
					strings.Repeat("02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n", 43) +
					"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
				So(actual, ShouldResemble, expected)
			})

			Convey("Long description and many events. both desc and events > msgLimit/2", func() {
				actual := formatter.Format(getParams(longEvents, moira.TriggerData{Desc: longDesc}, false))
				expected := "**NODATA**\n" +
					strings.Repeat("a", 1984) + "...\n" +
					"```\n" +
					strings.Repeat("02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n", 40) +
					"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```\n" +
					"...and 3 more events."
				So(actual, ShouldResemble, expected)
			})
		})
	})
}

func testBoldFormatter(str string) string {
	return fmt.Sprintf("**%s**", str)
}

func testDescriptionFormatter(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc += "\n"
	}
	return desc
}

func testUriFormatter(triggerURI, triggerName string) string {
	return fmt.Sprintf("[%s](%s)", triggerName, triggerURI)
}

func getParams(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) MessageFormatterParams {
	return MessageFormatterParams{
		Events:          events,
		Trigger:         trigger,
		MessageMaxChars: testMaxChars,
		Throttled:       throttled,
	}
}
