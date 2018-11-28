package index

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIndex_SearchTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	index := NewSearchIndex(logger, dataBase)

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

	Convey("First of all, fill index", t, func() {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil)

		err := index.fillIndex()
		index.indexed = true
		So(err, ShouldBeNil)
		docCount, _ := index.index.DocCount()
		So(docCount, ShouldEqual, uint64(31))
	})

	Convey("Search for triggers without pagination", t, func() {
		page := int64(0)
		size := int64(50)
		tags := make([]string, 0)
		searchString := ""
		onlyErrors := false

		Convey("No tags, no searchString, onlyErrors = false", func() {
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs)
			So(count, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, size = -1 (must return all triggers)", func() {
			size = -1
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs)
			So(count, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", func() {
			size = 50
			onlyErrors = true
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[:30])
			So(count, ShouldEqual, 30)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", func() {
			onlyErrors = true
			tags = []string{"encounters", "Kobold"}
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[1:3])
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", func() {
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[30:])
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("Empty list should be", func() {
			onlyErrors = true
			tags = []string{"Something-extremely-new"}
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldBeEmpty)
			So(count, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", func() {
			onlyErrors = true
			tags = make([]string, 0)
			searchString = "dragonshield medium"
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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
			actualTriggerIDs, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[:10])
			So(total, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 10", func() {
			page = 1
			actualTriggerIDs, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, triggerIDs[10:20])
			So(total, ShouldEqual, 31)
			So(err, ShouldBeNil)
		})

		Convey("No tags, no searchString, onlyErrors = false, page -> 1, size -> 20", func() {
			page = 1
			size = 20
			actualTriggerIDs, total, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, easyTriggerIDs)
			So(count, ShouldEqual, 3)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, search by name and description, 1 result", func() {
			searchString = "little monster"
			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
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

			actualTriggerIDs, count, err := index.SearchTriggers(tags, searchString, onlyErrors, page, size)
			So(actualTriggerIDs, ShouldResemble, mamaTrapsTriggerIDs)
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
		})
	})
}
