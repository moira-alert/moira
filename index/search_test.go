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
		searchOptions := moira.SearchOptions{
			Page:                  0,
			Size:                  50,
			OnlyProblems:          false,
			Tags:                  make([]string, 0),
			SearchString:          "",
			CreatedBy:             "",
			NeedSearchByCreatedBy: false,
		}

		Convey("No tags, no searchString, onlyErrors = false", func() {
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString))
			So(count, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, size = -1 (must return all triggers)", func() {
			searchOptions.Size = -1
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString))
			So(count, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", func() {
			searchOptions.Size = 50
			searchOptions.OnlyProblems = true
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[:30])
			So(count, ShouldEqual, 30)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", func() {
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"encounters", "Kobold"}
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[1:3])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", func() {
			searchOptions.OnlyProblems = false
			searchOptions.Tags = []string{"Something-extremely-new"}
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[30:])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("Empty list should be", func() {
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"Something-extremely-new"}
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldBeEmpty)
			So(count, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", func() {
			searchOptions.OnlyProblems = true
			searchOptions.Tags = make([]string, 0)
			searchOptions.SearchString = "dragonshield medium"
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[2:3])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms", func() {
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"traps"}
			searchOptions.SearchString = "deadly" //nolint

			deadlyTraps := []int{10, 14, 18, 19}

			deadlyTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range deadlyTraps {
				deadlyTrapsSearchResults = append(deadlyTrapsSearchResults, triggerTestCases.ToSearchResults(searchOptions.SearchString)[ind])
			}

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, deadlyTrapsSearchResults)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, no tags, no terms, with createdBy", func() {
			searchOptions.OnlyProblems = false
			searchOptions.Tags = make([]string, 0)
			searchOptions.SearchString = ""
			searchOptions.CreatedBy = "test"
			searchOptions.NeedSearchByCreatedBy = true

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[0:4])
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, one tag, no terms, with createdBy", func() {
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"shadows"}
			searchOptions.SearchString = ""
			searchOptions.CreatedBy = "tarasov.da"
			searchOptions.NeedSearchByCreatedBy = true

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[14:16])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, one text term, with createdBy", func() {
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"Coldness", "Dark"}
			searchOptions.SearchString = "deadly"
			searchOptions.CreatedBy = "tarasov.da"
			searchOptions.NeedSearchByCreatedBy = true

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[18:19])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, no text, with EMPTY createdBy", func() {
			searchOptions.OnlyProblems = true
			searchOptions.SearchString = ""
			searchOptions.Tags = []string{"Darkness", "DND-generator"}
			searchOptions.CreatedBy = ""
			searchOptions.NeedSearchByCreatedBy = true

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[5:7])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers with pagination", t, func() {
		searchOptions := moira.SearchOptions{
			Page:                  0,
			Size:                  10,
			OnlyProblems:          false,
			Tags:                  make([]string, 0),
			SearchString:          "",
			CreatedBy:             "",
			NeedSearchByCreatedBy: false,
		}

		Convey("No tags, no searchString, onlyErrors = false, page -> 0, size -> 10", func() {
			searchResults, total, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[:10])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 10", func() {
			searchOptions.Page = 1
			searchResults, total, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[10:20])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 20", func() {
			searchOptions.Page = 1
			searchOptions.Size = 20
			searchResults, total, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[20:])
			So(total, ShouldEqual, 32)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 0, size 2", func() {
			searchOptions.Page = 0
			searchOptions.Size = 2
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"traps"}
			searchOptions.SearchString = "deadly"

			deadlyTraps := []int{10, 14, 18, 19}

			deadlyTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range deadlyTraps {
				deadlyTrapsSearchResults = append(deadlyTrapsSearchResults, triggerTestCases.ToSearchResults(searchOptions.SearchString)[ind])
			}

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, deadlyTrapsSearchResults[:2])
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms, page -> 1, size 10", func() {
			searchOptions.Page = 1
			searchOptions.Size = 10
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"traps"}
			searchOptions.SearchString = "deadly"

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldBeEmpty)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, no terms, with createdBy, page -> 0, size 2", func() {
			searchOptions.Page = 0
			searchOptions.Size = 2
			searchOptions.OnlyProblems = true
			searchOptions.Tags = []string{"Human", "NPCs"}
			searchOptions.SearchString = ""
			searchOptions.CreatedBy = "internship2023"
			searchOptions.NeedSearchByCreatedBy = true

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[22:24])
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, one tags, no terms, with createdBy, page -> 0, size 5", func() {
			searchOptions.Page = 0
			searchOptions.Size = 5
			searchOptions.OnlyProblems = false
			searchOptions.Tags = []string{"Something-extremely-new"}
			searchOptions.SearchString = ""
			searchOptions.CreatedBy = "internship2023"
			searchOptions.NeedSearchByCreatedBy = true

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[30:32])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, no tags, no terms, with EMPTY createdBy, page -> 0, size 3", func() {
			searchOptions.Page = 0
			searchOptions.Size = 3
			searchOptions.OnlyProblems = false
			searchOptions.Tags = []string{}
			searchOptions.SearchString = ""
			searchOptions.CreatedBy = ""
			searchOptions.NeedSearchByCreatedBy = true

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[4:7])
			So(count, ShouldEqual, 3)
			So(err, ShouldBeNil)
		})
	})

	Convey("Search for triggers by description", t, func() {
		searchOptions := moira.SearchOptions{
			Page:                  0,
			Size:                  50,
			OnlyProblems:          false,
			Tags:                  make([]string, 0),
			SearchString:          "",
			CreatedBy:             "",
			NeedSearchByCreatedBy: false,
		}

		Convey("OnlyErrors = false, search by name and description, 0 results", func() {
			searchOptions.SearchString = "life female druid"
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldBeEmpty)
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 3 results", func() {
			searchOptions.SearchString = "easy"
			easy := []int{4, 9, 30, 31}

			easySearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range easy {
				easySearchResults = append(easySearchResults, triggerTestCases.ToSearchResults(searchOptions.SearchString)[ind])
			}

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, easySearchResults)
			So(count, ShouldEqual, 4)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 1 result", func() {
			searchOptions.SearchString = "little monster"
			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, triggerTestCases.ToSearchResults(searchOptions.SearchString)[4:5])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by description and tags, 2 results", func() {
			searchOptions.SearchString = "mama"
			searchOptions.Tags = []string{"traps"}

			mamaTraps := []int{11, 19}

			mamaTrapsSearchResults := make([]*moira.SearchResult, 0)
			for _, ind := range mamaTraps {
				mamaTrapsSearchResults = append(mamaTrapsSearchResults, triggerTestCases.ToSearchResults(searchOptions.SearchString)[ind])
			}

			searchResults, count, err := index.SearchTriggers(searchOptions)
			So(searchResults, ShouldResemble, mamaTrapsSearchResults)
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, search by description, no tags, with createdBy, 3 results", func() {
			searchOptions.SearchString = "mama"
			searchOptions.Tags = make([]string, 0)
			searchOptions.CreatedBy = "monster"
			searchOptions.NeedSearchByCreatedBy = true

			_, count, err := index.SearchTriggers(searchOptions)
			So(count, ShouldEqual, 3)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by description, no tags, with EMPTY createdBy, 1 result", func() {
			searchOptions.SearchString = "little monster"
			searchOptions.Tags = make([]string, 0)
			searchOptions.CreatedBy = ""
			searchOptions.NeedSearchByCreatedBy = true

			_, count, err := index.SearchTriggers(searchOptions)
			So(count, ShouldEqual, 1)
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
		searchOptions := moira.SearchOptions{
			Page:                  0,
			Size:                  50,
			OnlyProblems:          false,
			Tags:                  make([]string, 0),
			SearchString:          "",
			CreatedBy:             "",
			NeedSearchByCreatedBy: false,
		}

		actualTriggerIDs, total, err := index.SearchTriggers(searchOptions)
		So(actualTriggerIDs, ShouldBeEmpty)
		So(total, ShouldBeZeroValue)
		So(err, ShouldNotBeNil)
	})
}
