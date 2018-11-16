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
	defer index.Stop()

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

	Convey("First of all, start and fill index", t, func() {
		dataBase.EXPECT().GetTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil)

		err := index.Start()
		So(err, ShouldBeNil)
		docCount, _ := index.index.DocCount()
		So(docCount, ShouldEqual, uint64(31))
		So(index.IsReady(), ShouldBeTrue)
	})

	Convey("Search for triggers", t, func() {
		tags := make([]string, 0)
		textTerms := make([]string, 0)
		onlyErrors := false

		Convey("No tags, no textTerms, onlyErrors = false", func() {
			actual, err := index.SearchTriggers(tags, textTerms, onlyErrors)
			So(actual, ShouldResemble, triggerIDs)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true", func() {
			onlyErrors = true
			actual, err := index.SearchTriggers(tags, textTerms, onlyErrors)
			So(actual, ShouldResemble, triggerIDs[:30])
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags", func() {
			onlyErrors = true
			tags = []string{"encounters", "Kobold"}
			actual, err := index.SearchTriggers(tags, textTerms, onlyErrors)
			So(actual, ShouldResemble, triggerIDs[1:3])
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = false, several tags", func() {
			onlyErrors = false
			tags = []string{"Something-extremely-new"}
			actual, err := index.SearchTriggers(tags, textTerms, onlyErrors)
			So(actual, ShouldResemble, triggerIDs[30:])
			So(err, ShouldBeNil)
		})

		Convey("Empty list should be", func() {
			onlyErrors = true
			tags = []string{"Something-extremely-new"}
			actual, err := index.SearchTriggers(tags, textTerms, onlyErrors)
			So(actual, ShouldBeEmpty)
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, no tags, several text terms", func() {
			onlyErrors = true
			tags = make([]string, 0)
			textTerms = []string{"dragonshield", "medium"}
			actual, err := index.SearchTriggers(tags, textTerms, onlyErrors)
			So(actual, ShouldResemble, triggerIDs[2:3])
			So(err, ShouldBeNil)
		})

		Convey("OnlyErrors = true, several tags, several text terms", func() {
			onlyErrors = true
			tags = []string{"traps"}
			textTerms = []string{"deadly"}

			deadlyTrapsIDs := []string{
				triggerChecks[10].ID,
				triggerChecks[14].ID,
				triggerChecks[18].ID,
				triggerChecks[19].ID,
			}

			actual, err := index.SearchTriggers(tags, textTerms, onlyErrors)
			So(actual, ShouldResemble, deadlyTrapsIDs)
			So(err, ShouldBeNil)
		})
	})
}
