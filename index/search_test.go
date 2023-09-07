package index

import (
	"testing"

	"github.com/moira-alert/moira/metrics"

	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
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

	index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, fill index", t, func() {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)

		err := index.fillIndex()
		index.indexed = true
		So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		So(docCount, ShouldEqual, int64(32))
	})

	Convey("Search for triggers without pagination", t, func() {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false
		createdBy := ""

		Convey("No tags, no searchString, onlyErrors = false", func() {
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			So(count, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, size = -1 (must return all triggers)", func() {
			size = -1
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString))
			So(count, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", func() {
			size = 50
			onlyErrors = true
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:30])
			So(count, ShouldEqual, 30)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", func() {
			onlyErrors = true
			tags = []string{"encounters", "Kobold"}
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[1:3])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", func() {
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[30:])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("Empty list should be", func() {
			onlyErrors = true
			tags = []string{"Something-extremely-new"}
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldBeEmpty)
			So(count, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", func() {
			onlyErrors = true
			tags = make([]string, 0)
			searchString = "dragonshield medium"
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, deadlyTrapsSearchResults)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, no tags, no terms, with createdBy", func() {
			onlyErrors = false
			tags = make([]string, 0)
			searchString = ""
			createdBy = "test"

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[0:4])
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, one tag, no terms, with createdBy", func() {
			onlyErrors = true
			tags = []string{"shadows"}
			searchString = ""
			createdBy = "tarasov.da"

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[14:16])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, one text term, with createdBy", func() {
			onlyErrors = true
			tags = []string{"Coldness", "Dark"}
			searchString = "deadly"
			createdBy = "tarasov.da"

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[18:19])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers with pagination", t, func() {
		page := int64(0)
		size := int64(10)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false
		createdBy := ""

		Convey("No tags, no searchString, onlyErrors = false, page -> 0, size -> 10", func() {
			searchResults, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[:10])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 10", func() {
			page = 1
			searchResults, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[10:20])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 20", func() {
			page = 1
			size = 20
			searchResults, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldBeEmpty)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, no terms, with createdBy, page -> 0, size 2", func() {
			page = 0
			size = 2
			onlyErrors = true
			tags = []string{"Human", "NPCs"}
			searchString = ""
			createdBy = "internship2023"

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[22:24])
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, one tags, no terms, with createdBy, page -> 0, size 5", func() {
			page = 0
			size = 5
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			searchString = ""
			createdBy = "internship2023"

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[30:32])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers by description", t, func() {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false
		createdBy := ""

		Convey("OnlyErrors = false, search by name and description, 0 results", func() {
			searchString = "life female druid"
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
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

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, easySearchResults)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 1 result", func() {
			searchString = "little monster"
			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchString)[4:5])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by description and tags, 2 results", func() {
			searchString = "mama"
			tags = []string{"traps"}

			mamaTraps := []int{11, 19}

			mamaTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range mamaTraps {
				mamaTrapsSearchResults = append(mamaTrapsSearchResults, triggerTestCases.ToSearchResults(searchString)[ind])
			}

			searchResults, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(searchResults, ShouldResemble, mamaTrapsSearchResults)
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, search by description, no tags, with createdBy, 3 results", func() {
			searchString = "mama"
			tags = make([]string, 0)
			createdBy = "monster"

			_, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size, createdBy)
			So(count, ShouldEqual, 3)
			So(err, ShouldBeNil)
		})
	})
}

func TestIndex_SearchErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, fill index", t, func() {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)

		err := index.fillIndex()
		index.indexed = true
		So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		So(docCount, ShouldEqual, int64(32))
	})

	index.indexed = false

	Convey("Test search on non-ready index", t, func() {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		createdBy := ""

		actualTriggerIDs, total, err := index.SearchTriggers(tags, searchString, false, page, size, createdBy)
		So(actualTriggerIDs, ShouldBeEmpty)
		So(total, ShouldBeZeroValue)
		So(err, ShouldNotBeNil)
	})
}
