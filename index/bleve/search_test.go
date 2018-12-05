package bleve

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerIndex_Search(t *testing.T) {
	var newIndex *TriggerIndex
	var err error
	var count int64

	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	triggerIDs := make([]string, len(triggerChecks))
	for i, trigger := range triggerChecks {
		triggerIDs[i] = trigger.ID
	}

	triggersPointers := make([]*moira.TriggerCheck, len(triggerChecks))
	for i, trigger := range triggerChecks {
		newTrigger := new(moira.TriggerCheck)
		*newTrigger = trigger
		triggersPointers[i] = newTrigger
	}

	Convey("First of all, create and fill index", t, func() {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldBeZeroValue)
		So(err, ShouldBeNil)

		err = newIndex.Write(triggersPointers)
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldEqual, int64(31))
		So(err, ShouldBeNil)
	})

	Convey("Search for triggers without pagination", t, func() {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("No tags, no searchString, onlyErrors = false", func() {
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs)
			So(count, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, size = -1 (must return all triggers)", func() {
			size = -1
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs)
			So(count, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", func() {
			size = 50
			onlyErrors = true
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[:30])
			So(count, ShouldEqual, 30)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", func() {
			onlyErrors = true
			tags = []string{"encounters", "Kobold"}
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[1:3])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", func() {
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[30:])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("Empty list should be", func() {
			onlyErrors = true
			tags = []string{"Something-extremely-new"}
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldBeEmpty)
			So(count, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", func() {
			onlyErrors = true
			tags = make([]string, 0)
			searchString = "dragonshield medium"
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[2:3])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms", func() {
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly"

			deadlyTrapsIDs := []string{
				triggerChecks[10].ID,
				triggerChecks[14].ID,
				triggerChecks[18].ID,
				triggerChecks[19].ID,
			}

			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, deadlyTrapsIDs)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers with pagination", t, func() {
		page := int64(0)
		size := int64(10)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("No tags, no searchString, onlyErrors = false, page -> 0, size -> 10", func() {
			actualTriggerIDs, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[:10])
			So(total, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 10", func() {
			page = 1
			actualTriggerIDs, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[10:20])
			So(total, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 20", func() {
			page = 1
			size = 20
			actualTriggerIDs, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[20:])
			So(total, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 0, size 2", func() {
			page = 0
			size = 2
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly"

			deadlyTrapsIDs := []string{
				triggerChecks[10].ID,
				triggerChecks[14].ID,
				triggerChecks[18].ID,
				triggerChecks[19].ID,
			}

			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, deadlyTrapsIDs[:2])
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 1, size 10", func() {
			page = 1
			size = 10
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly"

			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldBeEmpty)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers by description", t, func() {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("OnlyErrors = false, search by name and description, 0 results", func() {
			searchString = "life female druid"
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldBeEmpty)
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 3 results", func() {
			easyTriggerIDs := []string{
				triggerChecks[4].ID,
				triggerChecks[9].ID,
				triggerChecks[30].ID,
			}

			searchString = "easy"
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, easyTriggerIDs)
			So(count, ShouldEqual, 3)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 1 result", func() {
			searchString = "little monster"
			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[4:5])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by description and tags, 2 results", func() {
			searchString = "mama"
			tags := []string{"traps"}

			mamaTrapsTriggerIDs := []string{
				triggerChecks[11].ID,
				triggerChecks[19].ID,
			}

			actualTriggerIDs, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, mamaTrapsTriggerIDs)
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})
	})

}

func TestStringsManipulations(t *testing.T) {
	Convey("Test escape symbols", t, func() {
		So(escapeString("12345"), ShouldResemble, "12345")
		So(escapeString("abcdefghijklmnop QRSTUVWXYZ"), ShouldResemble, "abcdefghijklmnop QRSTUVWXYZ")
		So(escapeString("I'm gonna use some.Bad.symbols.here. !!! Here we GO!!! (yeap, I mean it]"), ShouldResemble, "I m gonna use some Bad symbols here      Here we GO     yeap  I mean it ")
		So(escapeString(`+-=&|><!(){}[]^"'~*?\/`), ShouldResemble, "                      ")
	})

	Convey("Test split strings with symbols to escape", t, func() {
		So(splitStringToTerms("I.want.to.break:free!"), ShouldResemble, []string{"I", "want", "to", "break", "free"})
		So(splitStringToTerms("I;want-to,break_free!"), ShouldResemble, []string{"I", "want", "to", "break", "free"})
		So(splitStringToTerms(`I>want<to/break\free from&your@lies`), ShouldResemble, []string{"I", "want", "to", "break", "free", "from", "your", "lies"})
		So(splitStringToTerms(`(You)'[re] {so} "self" 'satisfied' |I| \don't/ ~need~ *you*`), ShouldResemble,
			[]string{"You", "re", "so", "self", "satisfied", "I", "don", "t", "need", "you"})
	})

	Convey("Test to split string in different languages", t, func() {
		So(splitStringToTerms("Привет, мир!"), ShouldResemble, []string{"Привет", "мир"})
		So(splitStringToTerms("Chào thế giới!"), ShouldResemble, []string{"Chào", "thế", "giới"})
		So(splitStringToTerms("ሰላም ልዑል!"), ShouldResemble, []string{"ሰላም", "ልዑል"})
		So(splitStringToTerms("Բարեւ աշխարհ!"), ShouldResemble, []string{"Բարեւ", "աշխարհ"})
		So(splitStringToTerms("ওহে বিশ্ব!"), ShouldResemble, []string{"ওহে", "বিশ্ব"})
		So(splitStringToTerms("你好 世界!"), ShouldResemble, []string{"你好", "世界"})
		So(splitStringToTerms("Γειά σου Κόσμε!"), ShouldResemble, []string{"Γειά", "σου", "Κόσμε"})
		So(splitStringToTerms("હેલો વર્લ્ડ!"), ShouldResemble, []string{"હેલો", "વર્લ્ડ"})
		So(splitStringToTerms("नमस्ते दुनिया!"), ShouldResemble, []string{"नमस्ते", "दुनिया"})
		So(splitStringToTerms("Გამარჯობა მსოფლიო!"), ShouldResemble, []string{"Გამარჯობა", "მსოფლიო"})
		So(splitStringToTerms("こんにちは世界!"), ShouldResemble, []string{"こんにちは世界"})
	})
}
