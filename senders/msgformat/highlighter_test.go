package msgformat

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
			DefaultDescriptionCutter,
			testBoldFormatter,
			testEventStringFormatter,
			"```",
			"```",
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
				"Please, **fix your system or tune this trigger** to generate less events."
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
			trigger.Desc = ""
			trigger.Tags = []string{}

			const (
				titleLine    = "**NODATA** [Name](http://moira.url/trigger/TriggerID)"
				eventLine    = "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
				endSuffix    = "...\n"
				lenEndSuffix = 4
			)

			lenTitle := utf8.RuneCountInString(titleLine) + len("\n") // 54 symbols
			oneEventLineLen := utf8.RuneCountInString(eventLine)      // 47 symbols

			var (
				msgLimit         = testMaxChars - lenTitle // 3947
				thirdOfLimit     = msgLimit / 3            // 1315
				greaterThanThird = thirdOfLimit + 100      // 1415
				lessThanThird    = thirdOfLimit - 100      // 1215
			)

			Convey("with long tags (tagsLen >= msgLimit), desc and events < msgLimit/3", func() {
				trigger.Tags = []string{
					strings.Repeat("a", 1000),
					strings.Repeat("b", 1000),
					strings.Repeat("c", 1000),
					strings.Repeat("d", 1000),
				}
				trigger.Desc = genDescByLimit(lessThanThird)
				events := genEventsByLimit(event, oneEventLineLen, lessThanThird)

				expected := titleLine +
					DefaultTagsLimiter(trigger.Tags,
						msgLimit-utf8.RuneCountInString(trigger.Desc)-len("```\n```")-oneEventLineLen*len(events),
					) + "\n" +
					strings.Repeat("a", lessThanThird) + "\n" +
					"```" +
					strings.Repeat(eventLine, len(events)) +
					"\n```"

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(utf8.RuneCountInString(actual), ShouldBeLessThanOrEqualTo, testMaxChars)
			})

			Convey("with description > msgLimit/3, tags and events < msgLimit/3, and sum of lengths is greater than msgLimit", func() {
				longDescLen := greaterThanThird + 200

				trigger.Tags = genTagsByLimit(lessThanThird)
				trigger.Desc = genDescByLimit(longDescLen)
				events := genEventsByLimit(event, oneEventLineLen, lessThanThird)

				tagsStr := " " + trigger.GetTags()

				expected := titleLine + tagsStr + "\n" +
					strings.Repeat("a",
						msgLimit-utf8.RuneCountInString(tagsStr)-len("```\n```")-oneEventLineLen*len(events)-lenEndSuffix,
					) + endSuffix +
					"```" +
					strings.Repeat(eventLine, len(events)) +
					"\n```"

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(utf8.RuneCountInString(actual), ShouldBeLessThanOrEqualTo, testMaxChars)
			})

			Convey("with long events string (> msgLimit/3), desc and tags < msgLimit/3", func() {
				longEventsLen := greaterThanThird + 200

				trigger.Tags = genTagsByLimit(lessThanThird)
				trigger.Desc = genDescByLimit(lessThanThird)
				events := genEventsByLimit(event, oneEventLineLen, longEventsLen)

				tagsStr := " " + trigger.GetTags()

				expected := titleLine + tagsStr + "\n" +
					strings.Repeat("a", lessThanThird) + "\n" +
					"```" +
					strings.Repeat(eventLine, 31) +
					"\n```\n" +
					"...and 3 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(utf8.RuneCountInString(actual), ShouldBeLessThanOrEqualTo, testMaxChars)
			})

			Convey("with tags and desc > msgLimit/3, events <= msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(greaterThanThird)
				trigger.Desc = genDescByLimit(greaterThanThird)
				events := genEventsByLimit(event, oneEventLineLen, lessThanThird)

				expected := titleLine + DefaultTagsLimiter(trigger.Tags, thirdOfLimit) + "\n" +
					strings.Repeat("a", greaterThanThird) + "\n" +
					"```" +
					strings.Repeat(eventLine, len(events)) +
					"\n```"

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(utf8.RuneCountInString(actual), ShouldBeLessThanOrEqualTo, testMaxChars)
			})

			Convey("with tags and events > msgLimit/3, desc <= msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(greaterThanThird)
				trigger.Desc = genDescByLimit(lessThanThird)
				events := genEventsByLimit(event, oneEventLineLen, greaterThanThird)

				expected := titleLine + DefaultTagsLimiter(trigger.Tags, thirdOfLimit) + "\n" +
					strings.Repeat("a", lessThanThird) + "\n" +
					"```" +
					strings.Repeat(eventLine, 29) +
					"\n```\n" + "...and 1 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(utf8.RuneCountInString(actual), ShouldBeLessThanOrEqualTo, testMaxChars)
			})

			Convey("with desc and events > msgLimit/3, tags <= msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(lessThanThird)
				trigger.Desc = genDescByLimit(greaterThanThird)
				events := genEventsByLimit(event, oneEventLineLen, greaterThanThird)

				tagsStr := DefaultTagsLimiter(trigger.Tags, lessThanThird)

				expected := titleLine + tagsStr + "\n" +
					strings.Repeat("a", thirdOfLimit+(thirdOfLimit-utf8.RuneCountInString(tagsStr))/2-lenEndSuffix) + endSuffix +
					"```" +
					strings.Repeat(eventLine, 28) +
					"\n```\n" + "...and 2 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(utf8.RuneCountInString(actual), ShouldBeLessThanOrEqualTo, testMaxChars)
			})

			Convey("tags, description and events all have len > msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(greaterThanThird)
				trigger.Desc = genDescByLimit(greaterThanThird)
				events := genEventsByLimit(event, oneEventLineLen, greaterThanThird)

				expected := titleLine + DefaultTagsLimiter(trigger.Tags, thirdOfLimit) + "\n" +
					strings.Repeat("a", thirdOfLimit-lenEndSuffix) + endSuffix +
					"```" +
					strings.Repeat(eventLine, thirdOfLimit/oneEventLineLen) +
					"\n```\n" +
					"...and 3 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(utf8.RuneCountInString(actual), ShouldBeLessThanOrEqualTo, testMaxChars)
			})
		})

		Convey("Contact extra message", func() {
			actual := formatter.Format(MessageFormatterParams{
				Events:          moira.NotificationEvents{event},
				Trigger:         trigger,
				MessageMaxChars: testMaxChars,
				Contact:         moira.ContactData{ExtraMessage: "@moira, help!"},
				Throttled:       false,
			})
			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				"@moira, help!" + "\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n```"
			So(actual, ShouldResemble, expected)
		})
	})
}

func genTagsByLimit(limit int) []string {
	tagName := "tag1"

	tagsCount := (limit - 1) / (len(tagName) + 2)

	tags := make([]string, 0, tagsCount)

	for i := 0; i < tagsCount; i++ {
		tags = append(tags, tagName)
	}

	return tags
}

func genDescByLimit(limit int) string {
	return strings.Repeat("a", limit)
}

func genEventsByLimit(event moira.NotificationEvent, oneEventLineLen int, limit int) moira.NotificationEvents {
	var events moira.NotificationEvents
	for i := 0; i < limit/oneEventLineLen; i++ {
		events = append(events, event)
	}

	return events
}

func testBoldFormatter(str string) string {
	return fmt.Sprintf("**%s**", str)
}

func testDescriptionFormatter(trigger moira.TriggerData, contact moira.ContactData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc += "\n"
	}

	if contact.ExtraMessage != "" {
		desc = contact.ExtraMessage + "\n" + desc
	}

	return desc
}

func testUriFormatter(triggerURI, triggerName string) string {
	return fmt.Sprintf("[%s](%s)", triggerName, triggerURI)
}

func testEventStringFormatter(event moira.NotificationEvent, location *time.Location) string {
	return fmt.Sprintf(
		"%s: %s = %s (%s to %s)",
		event.FormatTimestamp(location, moira.DefaultTimeFormat),
		event.Metric,
		event.GetMetricsValues(moira.DefaultNotificationSettings),
		event.OldState,
		event.State)
}

func getParams(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) MessageFormatterParams {
	return MessageFormatterParams{
		Events:          events,
		Trigger:         trigger,
		MessageMaxChars: testMaxChars,
		Throttled:       throttled,
	}
}
