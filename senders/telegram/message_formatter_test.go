package telegram

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/msgformat"

	. "github.com/smartystreets/goconvey/convey"
)

const testFrontURI = "https://moira.uri"

func TestMessageFormatter_Format(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	emojiProvider := telegramEmojiProvider{}

	formatter := NewTelegramMessageFormatter(
		emojiProvider,
		true,
		testFrontURI,
		location)

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

	expectedFirstLine := "💣 <b>NODATA</b> <a href=\"https://moira.uri/trigger/TriggerID\">Name</a> [tag1][tag2]\n"

	eventStr := "02:40 (GMT+00:00): <code>Metric</code> = 123 (OK to NODATA)\n"
	lenEventStr := utf8.RuneCountInString(eventStr) - utf8.RuneCountInString("<code></code>") // 60 - 13 = 47

	Convey("TelegramMessageFormatter", t, func() {
		Convey("message with one event", func() {
			events, throttled := moira.NotificationEvents{event}, false
			expected := expectedFirstLine +
				shortDesc + "\n" +
				eventsBlockStart + "\n" +
				eventStr +
				eventsBlockEnd

			msg := formatter.Format(getParams(events, trigger, throttled))

			So(msg, ShouldEqual, expected)
		})

		Convey("message with one event and throttled", func() {
			events, throttled := moira.NotificationEvents{event}, true
			msg := formatter.Format(getParams(events, trigger, throttled))

			expected := expectedFirstLine +
				shortDesc + "\n" +
				eventsBlockStart + "\n" +
				eventStr +
				eventsBlockEnd +
				throttleMsg
			So(msg, ShouldEqual, expected)
		})

		Convey("message with 3 events", func() {
			events, throttled := moira.NotificationEvents{event, event, event}, false
			expected := expectedFirstLine +
				shortDesc + "\n" +
				eventsBlockStart + "\n" +
				strings.Repeat(eventStr, 3) +
				eventsBlockEnd

			msg := formatter.Format(getParams(events, trigger, throttled))

			So(msg, ShouldEqual, expected)
		})

		Convey("message with complex description", func() {
			trigger.Desc = "# Моё описание\n\nсписок:\n- **жирный**\n- *курсив*\n- `код`\n- <u>подчёркнутый</u>\n- ~~зачёркнутый~~\n" +
				"\n------\nif a > b do smth\nif c < d do another thing\ntrue && false = false\ntrue || false = true\n" +
				"\"Hello everybody!\", 'another quots'\nif I use something like <custom_tag> nothing happens, also if i use allowed <b> tag"
			events, throttled := moira.NotificationEvents{event}, false

			expected := expectedFirstLine +
				"<b>Моё описание</b>\n\nсписок:\n- <strong>жирный</strong>\n- <em>курсив</em>\n- <code>код</code>\n- &lt;u&gt;подчёркнутый&lt;/u&gt;\n- <del>зачёркнутый</del>\n" +
				"\n\n\nif a &gt; b do smth\nif c &lt; d do another thing\ntrue &amp;&amp; false = false\ntrue || false = true\n" +
				"&quot;Hello everybody!&quot;, 'another quots'\nif I use something like &lt;custom_tag&gt; nothing happens, also if i use allowed &lt;b&gt; tag\n" +
				eventsBlockStart + "\n" +
				eventStr +
				eventsBlockEnd

			msg := formatter.Format(getParams(events, trigger, throttled))

			So(msg, ShouldEqual, expected)
		})

		Convey("with long messages", func() {
			const (
				titleWithoutTags = "💣 <b>NODATA</b> <a href=\"https://moira.uri/trigger/TriggerID\">Name</a>"
			)

			titleLen := utf8.RuneCountInString(titleWithoutTags) -
				utf8.RuneCountInString("<b></b><a href=\"https://moira.uri/trigger/TriggerID\"></a>") + len("\n") // 70 - 57 + 1 = 14

			msgLimit := albumCaptionMaxCharacters - titleLen // 1024 - 14 = 1010
			thirdOfMsgLimit := msgLimit / 3
			greaterThanThird := thirdOfMsgLimit + 150
			lessThanThird := thirdOfMsgLimit - 100

			// see genDescByLimit
			symbolAtEndOfDescription := ""
			if thirdOfMsgLimit%2 != 0 {
				symbolAtEndOfDescription = "i"
			}

			Convey("with tags > msgLimit/3, desc and events < msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(greaterThanThird + 200)
				trigger.Desc = genDescByLimit(lessThanThird)
				events := genEventsByLimit(event, lenEventStr, lessThanThird)

				expected := titleWithoutTags +
					msgformat.DefaultTagsLimiter(trigger.Tags,
						msgLimit-lessThanThird-len(events)*lenEventStr-len("\n")) + "\n" +
					strings.Repeat("<strong>ё</strong>ж", lessThanThird/2) + symbolAtEndOfDescription + "\n" +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, len(events)) + eventsBlockEnd

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})

			Convey("with desc > msgLimit/3, tags and events < msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(lessThanThird)
				trigger.Desc = genDescByLimit(greaterThanThird + 200)
				events := genEventsByLimit(event, lenEventStr, lessThanThird)

				expected := titleWithoutTags +
					msgformat.DefaultTagsLimiter(trigger.Tags, lessThanThird) + "\n" +
					tooLongDescMessage +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, len(events)) + eventsBlockEnd

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})

			Convey("with events > msgLimit/3, tags and desc < msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(lessThanThird)
				trigger.Desc = genDescByLimit(lessThanThird)
				events := genEventsByLimit(event, lenEventStr, greaterThanThird+200)

				expected := titleWithoutTags +
					msgformat.DefaultTagsLimiter(trigger.Tags, lessThanThird) + "\n" +
					strings.Repeat("<strong>ё</strong>ж", lessThanThird/2) + symbolAtEndOfDescription + "\n" +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, 10) + eventsBlockEnd +
					"\n...and 4 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})

			Convey("with tags and desc > msgLimit/3, events < msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(greaterThanThird)
				trigger.Desc = genDescByLimit(greaterThanThird)
				events := genEventsByLimit(event, lenEventStr, lessThanThird)

				expected := titleWithoutTags +
					msgformat.DefaultTagsLimiter(trigger.Tags, thirdOfMsgLimit) + "\n" +
					tooLongDescMessage +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, len(events)) + eventsBlockEnd

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})

			Convey("with tags and events > msgLimit / 3, desc < msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(greaterThanThird)
				trigger.Desc = genDescByLimit(lessThanThird)
				events := genEventsByLimit(event, lenEventStr, greaterThanThird)

				expected := titleWithoutTags +
					msgformat.DefaultTagsLimiter(trigger.Tags, thirdOfMsgLimit) + "\n" +
					strings.Repeat("<strong>ё</strong>ж", lessThanThird/2) + symbolAtEndOfDescription + "\n" +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, 8) + eventsBlockEnd +
					"\n...and 2 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})

			Convey("with desc and events > msgLimit / 3, tags < msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(lessThanThird)
				trigger.Desc = genDescByLimit(greaterThanThird)
				events := genEventsByLimit(event, lenEventStr, greaterThanThird)

				expected := titleWithoutTags +
					msgformat.DefaultTagsLimiter(trigger.Tags, lessThanThird) + "\n" +
					tooLongDescMessage +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, 7) + eventsBlockEnd +
					"\n...and 3 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})

			Convey("with tags, desc, events > msgLimit/3", func() {
				trigger.Tags = genTagsByLimit(greaterThanThird)
				trigger.Desc = genDescByLimit(greaterThanThird)
				events := genEventsByLimit(event, lenEventStr, greaterThanThird)

				expected := titleWithoutTags +
					msgformat.DefaultTagsLimiter(trigger.Tags, thirdOfMsgLimit) + "\n" +
					tooLongDescMessage +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, 6) +
					eventsBlockEnd +
					"\n...and 4 more events."

				actual := formatter.Format(getParams(events, trigger, false))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})
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
	str := strings.Repeat("**ё**ж", limit/2)
	if limit%2 != 0 {
		str += "i"
	}
	return str
}

func genEventsByLimit(event moira.NotificationEvent, oneEventLineLen int, limit int) moira.NotificationEvents {
	var events moira.NotificationEvents
	for i := 0; i < limit/oneEventLineLen; i++ {
		events = append(events, event)
	}
	return events
}

func getParams(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) msgformat.MessageFormatterParams {
	return msgformat.MessageFormatterParams{
		Events:          events,
		Trigger:         trigger,
		MessageMaxChars: albumCaptionMaxCharacters,
		Throttled:       throttled,
	}
}
