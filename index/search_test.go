package index

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/fixtures"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

func TestIndex_SearchTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	index := NewSearchIndex(logger, dataBase)

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, fill index", t, func(c C) {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)

		err := index.fillIndex()
		index.indexed = true
		c.So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(32))
	})

	Convey("Search for triggers without pagination", t, func(c C) {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("No tags, no searchString, onlyErrors = false", t, func(c C) {
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			c.So(count, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, size = -1 (must return all triggers)", t, func(c C) {
			size = -1
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			c.So(count, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", t, func(c C) {
			size = 50
			onlyErrors = true
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:30])
			c.So(count, ShouldEqual, 30)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", t, func(c C) {
			onlyErrors = true
			tags = []string{"encounters", "Kobold"}
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[1:3])
			c.So(count, ShouldEqual, 2)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", t, func(c C) {
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[30:])
			c.So(count, ShouldEqual, 2)
			c.So(err, ShouldBeNil)
		})

		Convey("Empty list should be", t, func(c C) {
			onlyErrors = true
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldBeEmpty)
			c.So(count, ShouldBeZeroValue)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", t, func(c C) {
			onlyErrors = true
			tags = make([]string, 0)
			searchString = "dragonshield medium"
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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
			searchResults, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:10])
			c.So(total, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 10", t, func(c C) {
			page = 1
			searchResults, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[10:20])
			c.So(total, ShouldEqual, 32)
			c.So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 20", t, func(c C) {
			page = 1
			size = 20
			searchResults, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, easySearchResults)
			c.So(count, ShouldEqual, 4)
			c.So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 1 result", t, func(c C) {
			searchString = "little monster"
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			c.So(searchResults, ShouldResemble, mamaTrapsSearchResults)
			c.So(count, ShouldEqual, 2)
			c.So(err, ShouldBeNil)
		})
	})
}

func TestIndex_SearchErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	index := NewSearchIndex(logger, dataBase)

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, fill index", t, func(c C) {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)

		err := index.fillIndex()
		index.indexed = true
		c.So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(32))
	})

	index.indexed = false

	Convey("Test search on non-ready index", t, func(c C) {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""

		actualTriggerIDs, total, err := index.SearchTriggers(tags, searchString, false, page, size)
		c.So(actualTriggerIDs, ShouldBeEmpty)
		c.So(total, ShouldBeZeroValue)
		c.So(err, ShouldNotBeNil)
	})
}
