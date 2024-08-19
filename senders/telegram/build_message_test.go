package telegram

import (
	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestTelegramDescriptionFormatter(t *testing.T) {
	const shortDesc = "# Моё описание\n\nсписок:\n- **жирный**\n- *курсив*\n- `код`\n- <u>подчёркнутый</u>\n- ~~зачёркнутый~~\n\nif a > b do smth\nif c < d do another thing\ntrue && false = false\ntrue || false = true\n\"Hello everybody!\""
	trigger := moira.TriggerData{
		Tags: []string{"tag1", "tag2"},
		Name: "Name",
		ID:   "TriggerID",
		Desc: shortDesc,
	}

	Convey("Telegram description formatter", t, func() {
		Convey("with short description", func() {
			expected := "<b>Моё описание</b>\n\nсписок:\n- <b>жирный</b>\n- <i>курсив</i>\n- <code>код</code>\n- <u>подчёркнутый</u>\n- <s>зачёркнутый</s>\n\nif a &gt; b do smth\nif c &lt; d do another thing\ntrue &amp;&amp; false = false\ntrue || false = true\n&quot;Hello everybody!&quot;\n"

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
