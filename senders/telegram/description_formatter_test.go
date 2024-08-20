package telegram

import (
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

		//Convey("with unclosed text formatting tags", func() {
		//	const (
		//		introText  = "intro "
		//		tagContent = "text."
		//	)
		//
		//	markdownTags := []string{"**", "*", "`", "~~"}
		//	htmlStartTags := []string{"<b>", "<i>", "<code>", "<s>"}
		//	htmlEndTags := []string{"</b>", "</i>", "</code>", "</s>"}
		//
		//	for tagIndex := range markdownTags {
		//		Convey(fmt.Sprintf("%s tag", htmlStartTags[tagIndex]), func() {
		//			trigger.Desc = introText + markdownTags[tagIndex] + tagContent + markdownTags[tagIndex]
		//			fullExpected := introText + htmlStartTags[tagIndex] + tagContent + htmlEndTags[tagIndex] + "\n"
		//
		//			for maxSize := len(fullExpected); maxSize >= len(introText); maxSize -= 1 {
		//				Convey(fmt.Sprintf("with maxSize = %v", maxSize), func() {
		//					desc := descriptionFormatter(trigger, maxSize)
		//
		//					expected := fullExpected
		//					if maxSize != len(fullExpected) {
		//						if maxSize <= len(introText)+len(htmlStartTags[tagIndex]) {
		//							cutForSuffix := maxSize - len(introText) - len(endSuffix)
		//							if cutForSuffix > 0 {
		//								expected = introText[:len(introText)-cutForSuffix] + endSuffix
		//							} else {
		//								expected = introText + endSuffix
		//							}
		//						} else {
		//							tailLen := maxSize - len(introText) - len(htmlStartTags[tagIndex])
		//							if tailLen > len(htmlEndTags[tagIndex])+len(endSuffix) {
		//								forTagContent := tailLen - len(htmlEndTags[tagIndex]) - len(endSuffix)
		//								expected = introText + htmlStartTags[tagIndex] + tagContent[:forTagContent] + htmlEndTags[tagIndex] + endSuffix
		//							} else {
		//								expected = introText + endSuffix
		//							}
		//						}
		//					}
		//					So(desc, ShouldEqual, expected)
		//				})
		//			}
		//		})
		//	}
		//})
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
					start:    0,
					nodeType: openTag,
				},
				{
					content:  []rune("Моё описание"),
					start:    3,
					nodeType: text,
				},
				{
					content:  []rune("</b>"),
					start:    15,
					nodeType: closeTag,
				},
				{
					content:  []rune("\nif a "),
					start:    19,
					nodeType: text,
				},
				{
					content:  []rune("&gt;"),
					start:    25,
					nodeType: escapedSymbol,
				},
				{
					content:  []rune(" b do smth\n"),
					start:    29,
					nodeType: text,
				},
				{
					content:  []rune("<a href=\"http://example.com\">"),
					start:    40,
					nodeType: openTag,
				},
				{
					content:  []rune("текст ссылки"),
					start:    69,
					nodeType: text,
				},
				{
					content:  []rune("</a>"),
					start:    81,
					nodeType: closeTag,
				},
			}
			expectedUnclosed := []int{}
			expectedMaxSize := testMaxSize

			nodes, unclosed, maxSize := splitDescriptionIntoNodes([]rune(desc), testMaxSize)

			So(nodes, ShouldResemble, expectedNodes)
			So(unclosed, ShouldResemble, expectedUnclosed)
			So(maxSize, ShouldEqual, expectedMaxSize)
		})
	})
}
