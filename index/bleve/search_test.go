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

	Convey("First of all, create and fill index", t, func(c C) {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		c.So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		c.So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		c.So(count, ShouldBeZeroValue)
		c.So(err, ShouldBeNil)

		err = newIndex.Write(triggerChecksPointers)
		c.So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		c.So(count, ShouldEqual, int64(32))
		c.So(err, ShouldBeNil)
	})

	Convey("Search for triggers without pagination", t, func(c C) {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("No tags, no searchString, onlyErrors = false", t, func(c C) {
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			c.So(count, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, size = -1 (must return all triggers)", t, func(c C) {
			size = -1
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			c.So(count, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", t, func(c C) {
			size = 50
			onlyErrors = true
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:30])
			c.So(count, ShouldEqual, 30)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", t, func(c C) {
			onlyErrors = true
			tags = []string{"encounters", "Kobold"}
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[1:3])
			c.So(count, ShouldEqual, 2)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", t, func(c C) {
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[30:])
			c.So(count, ShouldEqual, 2)
			c.So(err, ShouldBeNil)
		})

		Convey("Empty list should be", t, func(c C) {
			onlyErrors = true
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldBeEmpty)
			c.So(count, ShouldBeZeroValue)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", t, func(c C) {
			onlyErrors = true
			tags = make([]string, 0)
			searchString = "dragonshield medium"
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[2:3])
			c.So(count, ShouldEqual, 1)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms", t, func(c C) {
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly"

			deadlyTraps := []int{10, 14, 18, 19}

			deadlyTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range deadlyTraps {
				deadlyTrapsSearchResults = append(deadlyTrapsSearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, deadlyTrapsSearchResults)
			c.So(count, ShouldEqual, 4)
			c.So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers with pagination", t, func(c C) {
		page := int64(0)
		size := int64(10)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("No tags, no searchString, onlyErrors = false, page -> 0, size -> 10", t, func(c C) {
			searchResults, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:10])
			c.So(total, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 10", t, func(c C) {
			page = 1
			searchResults, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[10:20])
			c.So(total, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 20", t, func(c C) {
			page = 1
			size = 20
			searchResults, total, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[20:])
			c.So(total, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 0, size 2", t, func(c C) {
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
			c.So(searchResults, ShouldResemble, deadlyTrapsSearchResults[:2])
			c.So(count, ShouldEqual, 4)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 1, size 10", t, func(c C) {
			page = 1
			size = 10
			onlyErrors = true
			tags = []string{"traps"}
			searchString = "deadly"

			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldBeEmpty)
			c.So(count, ShouldEqual, 4)
			c.So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers by description", t, func(c C) {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("OnlyErrors = false, search by name and description, 0 results", t, func(c C) {
			searchString = "life female druid"
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldBeEmpty)
			c.So(count, ShouldEqual, 0)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 3 results", t, func(c C) {
			searchString = "easy"
			easy := []int{4, 9, 30, 31}

			easySearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range easy {
				easySearchResults = append(easySearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchString = "easy"
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, easySearchResults)
			c.So(count, ShouldEqual, 4)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 1 result", t, func(c C) {
			searchString = "little monster"
			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[4:5])
			c.So(count, ShouldEqual, 1)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by description and tags, 2 results", t, func(c C) {
			searchString = "mama"
			tags := []string{"traps"}

			mamaTraps := []int{11, 19}

			mamaTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range mamaTraps {
				mamaTrapsSearchResults = append(mamaTrapsSearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchResults, count, err := newIndex.Search(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, mamaTrapsSearchResults)
			c.So(count, ShouldEqual, 2)
			c.So(err, ShouldBeNil)
		})
	})

}

func TestStringsManipulations(t *testing.T) {
	Convey("Test escape symbols", t, func(c C) {
		c.So(escapeString("12345"), ShouldResemble, "12345")
		c.So(escapeString("abcdefghijklmnop QRSTUVWXYZ"), ShouldResemble, "abcdefghijklmnop QRSTUVWXYZ")
		c.So(escapeString("I'm gonna use some.Bad.symbols.here. !!! Here we GO!!! (yeap, I mean it]"), ShouldResemble, "I m gonna use some Bad symbols here      Here we GO     yeap  I mean it ")
		c.So(escapeString(`+-=&|><!(){}[]^"'~*?\/`), ShouldResemble, "                      ")
	})

	Convey("Test split strings with symbols to escape", t, func(c C) {
		c.So(splitStringToTerms("I.want.to.break:free!"), ShouldResemble, []string{"I", "want", "to", "break", "free"})
		c.So(splitStringToTerms("I;want-to,break_free!"), ShouldResemble, []string{"I", "want", "to", "break", "free"})
		c.So(splitStringToTerms(`I>want<to/break\free from&your@lies`), ShouldResemble, []string{"I", "want", "to", "break", "free", "from", "your", "lies"})
		c.So(splitStringToTerms(`(You)'[re] {so} "self" 'satisfied' |I| \don't/ ~need~ *you*`), ShouldResemble,
			[]string{"You", "re", "so", "self", "satisfied", "I", "don", "t", "need", "you"})
	})

	Convey("Test to split string in different languages", t, func(c C) {
		c.So(splitStringToTerms("Привет, мир!"), ShouldResemble, []string{"Привет", "мир"})
		c.So(splitStringToTerms("Chào thế giới!"), ShouldResemble, []string{"Chào", "thế", "giới"})
		c.So(splitStringToTerms("ሰላም ልዑል!"), ShouldResemble, []string{"ሰላም", "ልዑል"})
		c.So(splitStringToTerms("Բարեւ աշխարհ!"), ShouldResemble, []string{"Բարեւ", "աշխարհ"})
		c.So(splitStringToTerms("ওহে বিশ্ব!"), ShouldResemble, []string{"ওহে", "বিশ্ব"})
		c.So(splitStringToTerms("你好 世界!"), ShouldResemble, []string{"你好", "世界"})
		c.So(splitStringToTerms("Γειά σου Κόσμε!"), ShouldResemble, []string{"Γειά", "σου", "Κόσμε"})
		c.So(splitStringToTerms("હેલો વર્લ્ડ!"), ShouldResemble, []string{"હેલો", "વર્લ્ડ"})
		c.So(splitStringToTerms("नमस्ते दुनिया!"), ShouldResemble, []string{"नमस्ते", "दुनिया"})
		c.So(splitStringToTerms("Გამარჯობა მსოფლიო!"), ShouldResemble, []string{"Გამარჯობა", "მსოფლიო"})
		c.So(splitStringToTerms("こんにちは世界!"), ShouldResemble, []string{"こんにちは世界"})
	})
}
