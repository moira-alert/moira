package bleve

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/fixtures"
	"github.com/moira-alert/moira/index/mapping"
)

func TestTriggerIndex_Search(t *testing.T) {
	var newIndex *TriggerIndex
	var err error
	var count int64

	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, create and fill index", t, func() {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldBeZeroValue)
		So(err, ShouldBeNil)

		err = newIndex.Write(triggerChecksPointers)
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldEqual, int64(32))
		So(err, ShouldBeNil)
	})

	Convey("Search for triggers without pagination", t, func() {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("No tags, no searchString, onlyErrors = false", func() {
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			So(count, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, size = -1 (must return all triggers)", func() {
			size = -1
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			So(count, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", func() {
			size = 50
			onlyErrors = true
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:30])
			So(count, ShouldEqual, 30)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", func() {
			onlyErrors = true
			tags = []string{"encounters", "Kobold"}
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[1:3])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", func() {
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[30:])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("Empty list should be", func() {
			onlyErrors = true
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldBeEmpty)
			So(count, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", func() {
			onlyErrors = true
			tags = make([]string, 0)
			searchString = "dragonshield medium"
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[2:3])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms", func() {
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly" //nolint

			deadlyTraps := []int{10, 14, 18, 19}

			deadlyTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range deadlyTraps {
				deadlyTrapsSearchResults = append(deadlyTrapsSearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, deadlyTrapsSearchResults)
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
			searchResults, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:10])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 10", func() {
			page = 1
			searchResults, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[10:20])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 20", func() {
			page = 1
			size = 20
			searchResults, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[20:])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 0, size 2", func() {
			page = 0
			size = 2
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly"

			deadlyTraps := []int{10, 14, 18, 19}

			deadlyTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range deadlyTraps {
				deadlyTrapsSearchResults = append(deadlyTrapsSearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, deadlyTrapsSearchResults[:2])
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 1, size 10", func() {
			page = 1
			size = 10
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly"

			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldBeEmpty)
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
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldBeEmpty)
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 3 results", func() {
			searchString = "easy"
			easy := []int{4, 9, 30, 31}

			easySearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range easy {
				easySearchResults = append(easySearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchString = "easy"
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, easySearchResults)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 1 result", func() {
			searchString = "little monster"
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[4:5])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by description and tags, 2 results", func() {
			searchString = "mama"
			tags := []string{"traps"}

			mamaTraps := []int{11, 19}

			mamaTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range mamaTraps {
				mamaTrapsSearchResults = append(mamaTrapsSearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			So(searchResults, ShouldResemble, mamaTrapsSearchResults)
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})
	})

	Convey("Test close index", t, func() {
		err := newIndex.Close()
		So(err, ShouldBeNil)
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
