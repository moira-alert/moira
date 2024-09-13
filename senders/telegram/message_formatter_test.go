package telegram

import (
	"fmt"
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
	lenFirstLine := utf8.RuneCountInString(expectedFirstLine) -
		utf8.RuneCountInString("<b></b><a href=\"https://moira.uri/trigger/TriggerID\"></a>") // 83 - 57

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
			msgLimit := albumCaptionMaxCharacters - lenFirstLine
			halfMsgLimit := msgLimit / 2
			greaterThanHalf := halfMsgLimit + 100
			lessThanHalf := halfMsgLimit - 100

			Convey("text size of description > msgLimit / 2", func() {
				var events moira.NotificationEvents
				throttled := false

				eventsCount := lessThanHalf / lenEventStr
				for i := 0; i < eventsCount; i++ {
					events = append(events, event)
				}

				trigger.Desc = strings.Repeat("**ё**ж", greaterThanHalf/2)

				expected := expectedFirstLine +
					strings.Repeat("<strong>ё</strong>ж", greaterThanHalf/2) + "\n" +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, eventsCount) +
					eventsBlockEnd

				msg := formatter.Format(getParams(events, trigger, throttled))

				So(calcRunesCountWithoutHTML(msg), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
				So(msg, ShouldEqual, expected)
			})

			Convey("text size of events block > msgLimit / 2", func() {
				var events moira.NotificationEvents
				throttled := false

				eventsCount := greaterThanHalf / lenEventStr
				for i := 0; i < eventsCount; i++ {
					events = append(events, event)
				}

				trigger.Desc = strings.Repeat("**ё**ж", lessThanHalf/2)

				expected := expectedFirstLine +
					strings.Repeat("<strong>ё</strong>ж", lessThanHalf/2) + "\n" +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, eventsCount) +
					eventsBlockEnd

				msg := formatter.Format(getParams(events, trigger, throttled))

				So(calcRunesCountWithoutHTML(msg), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
				So(msg, ShouldEqual, expected)
			})

			Convey("both description and events block have text size > msgLimit/2", func() {
				var events moira.NotificationEvents
				throttled := false

				eventsCount := greaterThanHalf / lenEventStr
				for i := 0; i < eventsCount; i++ {
					events = append(events, event)
				}

				trigger.Desc = strings.Repeat("**ё**ж", greaterThanHalf/2)

				eventsShouldBe := halfMsgLimit / lenEventStr

				expected := expectedFirstLine +
					tooLongDescMessage +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, eventsShouldBe) +
					eventsBlockEnd +
					fmt.Sprintf("\n...and %d more events.", len(events)-eventsShouldBe)

				msg := formatter.Format(getParams(events, trigger, throttled))

				So(calcRunesCountWithoutHTML(msg), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
				So(msg, ShouldEqual, expected)
			})

			Convey("with tags, desc, events > msgLimit/3", func() {
				var (
					events           moira.NotificationEvents
					throttled        = false
					greaterThanThird = albumCaptionMaxCharacters/3 + 100
				)

				tag := "tag1"
				tagCount := (greaterThanThird - len(" ")) / (len(tag) + len("[]"))

				tags := make([]string, 0, tagCount)
				for i := 0; i < tagCount; i++ {
					tags = append(tags, tag)
				}
				trigger.Tags = tags

				trigger.Desc = strings.Repeat("**ё**ж", greaterThanThird/2)

				eventsCount := greaterThanThird / lenEventStr
				for i := 0; i < eventsCount; i++ {
					events = append(events, event)
				}

				expected := "💣 <b>NODATA</b> <a href=\"https://moira.uri/trigger/TriggerID\">Name</a>" +
					msgformat.DefaultTagsLimiter(trigger.Tags, (albumCaptionMaxCharacters-(lenFirstLine-len(" [tag1][tag2]")))/3) + "\n" +
					tooLongDescMessage +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, 6) +
					eventsBlockEnd +
					"\n...and 3 more events."

				actual := formatter.Format(getParams(events, trigger, throttled))

				So(actual, ShouldResemble, expected)
				So(calcRunesCountWithoutHTML(actual), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
			})
		})
	})
}

func getParams(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) msgformat.MessageFormatterParams {
	return msgformat.MessageFormatterParams{
		Events:          events,
		Trigger:         trigger,
		MessageMaxChars: albumCaptionMaxCharacters,
		Throttled:       throttled,
	}
}
