package telegram

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/msgformat"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	testFrontURI = "http://moira.uri"
)

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{
		formatter: msgformat.NewHighlightSyntaxFormatter(
			telegramEmojiProvider{},
			true,
			testFrontURI,
			location,
			urlFormatter,
			descriptionFormatter,
			boldFormatter,
			eventStringFormatter,
			codeBlockStart,
			codeBlockEnd),
	}

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

	Convey("Telegram sender with configured formatter", t, func() {
		Convey("Message is html formatted", func() {
			events, throttled := moira.NotificationEvents{event}, true
			msg := sender.buildMessage(events, trigger, throttled, albumCaptionMaxCharacters)

			expected := "💣<b>NODATA</b> <a href=\"http://moira.uri/trigger/TriggerID\">Name</a> [tag1][tag2]\n" +
				shortDesc + "\n" +
				codeBlockStart + "\n" +
				"02:40 (GMT+00:00): <code>Metric</code> = 123 (OK to NODATA)\n" +
				codeBlockEnd +
				"Please, <b>fix your system or tune this trigger</b> to generate less events."
			So(msg, ShouldEqual, expected)
		})
	})
}
