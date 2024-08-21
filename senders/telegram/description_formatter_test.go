package telegram

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTelegramDescriptionFormatter(t *testing.T) {
	const (
		shortDesc    = "# Моё описание\n\nсписок:\n- **жирный**\n- *курсив*\n- `код`\n- <u>подчёркнутый</u>\n- ~~зачёркнутый~~\n\nif a > b do smth\nif c < d do another thing\ntrue && false = false\ntrue || false = true\n\"Hello everybody!\""
		expectedDesc = "<b>Моё описание</b>\n\nсписок:\n- <b>жирный</b>\n- <i>курсив</i>\n- <code>код</code>\n- <u>подчёркнутый</u>\n- <s>зачёркнутый</s>\n\nif a &gt; b do smth\nif c &lt; d do another thing\ntrue &amp;&amp; false = false\ntrue || false = true\n&quot;Hello everybody!&quot;\n"
	)

	trigger := moira.TriggerData{
		Tags: []string{"tag1", "tag2"},
		Name: "Name",
		ID:   "TriggerID",
		Desc: shortDesc,
	}

	Convey("Telegram description formatter", t, func() {
		Convey("with short description", func() {
			expected := expectedDesc

			desc := descriptionFormatter(trigger, -1)

			So(desc, ShouldEqual, expected)
		})
	})
}

func TestSplitDescriptionIntoNodes(t *testing.T) {
	Convey("Split description", t, func() {
		Convey("with no unclosed tags", func() {
			desc := "<b>Моё описание</b>\nif a &gt; b do smth\n<a href=\"http://example.com\">текст ссылки</a>"
			testMaxSize := len([]rune(desc))

			expectedNodes := []descriptionNode{
				{
					content:  []rune("<b>"),
					nodeType: openTag,
				},
				{
					content:  []rune("Моё описание"),
					nodeType: text,
				},
				{
					content:  []rune("</b>"),
					nodeType: closeTag,
				},
				{
					content:  []rune("\nif a "),
					nodeType: text,
				},
				{
					content:  []rune("&gt;"),
					nodeType: escapedSymbol,
				},
				{
					content:  []rune(" b do smth\n"),
					nodeType: text,
				},
				{
					content:  []rune("<a href=\"http://example.com\">"),
					nodeType: openTag,
				},
				{
					content:  []rune("текст ссылки"),
					nodeType: text,
				},
				{
					content:  []rune("</a>"),
					nodeType: closeTag,
				},
			}
			expectedUnclosed := []int{}

			nodes, unclosed := splitDescriptionIntoNodes([]rune(desc), testMaxSize)

			So(nodes, ShouldResemble, expectedNodes)
			So(unclosed, ShouldResemble, expectedUnclosed)
		})
	})
}

func TestCutDescription(t *testing.T) {
	Convey("Cut description", t, func() {
		type testCase struct {
			caseDesc string
			desc     string
			maxSize  int
			expected string
		}

		prepared := []testCase{
			{
				caseDesc: "no unclosed tags",
				desc:     "абра<b>кадабра</b>один",
				maxSize:  20,
				expected: "абра<b>кадабра</b>од",
			},
			{
				caseDesc: "need to close tags",
				desc:     "абра<i>кадабра</i>один",
				maxSize:  13,
				expected: "абра<i>ка</i>",
			},
			{
				caseDesc: "after closing tags need to remove empty tag pair",
				desc:     "абра<s>кадабра</s>один",
				maxSize:  11,
				expected: "абра",
			},
			{
				caseDesc: "after closing tags need to remove all empty tag pairs",
				desc:     "абра<s><code><b>кадабра</b></code></s>один",
				maxSize:  19,
				expected: "абра",
			},
			{
				caseDesc: "close nested tags",
				desc:     "абра<code><u>кадабра abra</u></code>один",
				maxSize:  27,
				expected: "абра<code><u>кад</u></code>",
			},
			{
				caseDesc: "with unclosed <pre> tag",
				desc:     "абра<b><pre>\nfunc hello() {\n\tfmt.Printf(\"hello\")\n}\n</pre><b>",
				maxSize:  16,
				expected: "абра",
			},
			{
				caseDesc: "with link, first remove tags from short name",
				desc:     "теxt: <a href=\"http://example.com\"><b><u>cсылка</u></b></a>",
				maxSize:  47,
				expected: "теxt: <a href=\"http://example.com\">cсылка</a>",
			},
			{
				caseDesc: "with link, first remove tags from short name, then cut short name",
				desc:     "теxt: <a href=\"http://example.com\"><b><u>cсылка</u></b></a>",
				maxSize:  45,
				expected: "теxt: <a href=\"http://example.com\">cсыл</a>",
			},
			{
				caseDesc: "with link, but link should be cut entirely",
				desc:     "теxt: <a href=\"http://example.com\"><b><u>cсылка</u></b></a>",
				maxSize:  38,
				expected: "теxt: ",
			},
			{
				caseDesc: "with escaped symbols, such symbols are cut entirely",
				desc:     "абра<blockquote>if a &gt; or c &lt; d do smth</blockquote>",
				maxSize:  37,
				expected: "абра<blockquote>if a </blockquote>",
			},
		}

		for i := range prepared {
			Convey(fmt.Sprintf("case %v: %s (maxSize = %v)", i+1, prepared[i].caseDesc, prepared[i].maxSize), func() {
				desc := cutDescription([]rune(prepared[i].desc), prepared[i].maxSize)

				So(len([]rune(desc)), ShouldBeLessThanOrEqualTo, prepared[i].maxSize)
				So(desc, ShouldEqual, prepared[i].expected)
			})
		}
	})
}
