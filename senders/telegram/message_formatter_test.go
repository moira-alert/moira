package telegram

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/msgformat"
	"strings"
	"testing"
	"time"

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
	//firstLineLen := len([]rune(expectedFirstLine)) - len([]rune("<b></b><a href=\"https://moira.uri/trigger/TriggerID\"></a>"))

	eventStr := "02:40 (GMT+00:00): <code>Metric</code> = 123 (OK to NODATA)\n"
	lenEventStr := len([]rune(eventStr)) - len([]rune("<code></code>"))

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
			msgLimit := albumCaptionMaxCharacters
			halfMsgLimit := msgLimit / 2
			greaterThanHalf := halfMsgLimit + 100
			lessThanHalf := halfMsgLimit - 100

			Convey("text size of description > msgLimit / 2", func() {
				var events moira.NotificationEvents
				eventsCount := lessThanHalf / lenEventStr
				for i := 0; i < eventsCount; i++ {
					events = append(events, event)
				}
				throttled := false

				trigger.Desc = strings.Repeat("**ё**ж", greaterThanHalf/2)

				expected := expectedFirstLine +
					strings.Repeat("<strong>ё</strong>ж", 306) + "\n" +
					eventsBlockStart + "\n" +
					strings.Repeat(eventStr, 8) +
					eventsBlockEnd

				msg := formatter.Format(getParams(events, trigger, throttled))

				So(calcRunesCountWithoutHTML([]rune(msg)), ShouldBeLessThanOrEqualTo, albumCaptionMaxCharacters)
				So(msg, ShouldEqual, expected)
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
