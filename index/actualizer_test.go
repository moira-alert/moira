package index

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIndex_actualize(t *testing.T) {
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

	Convey("First of all, start and fill index", t, func() {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs[:20], nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs[:20]).Return(triggersPointers[:20], nil)

		err := index.Start()
		So(err, ShouldBeNil)
		docCount, _ := index.index.DocCount()
		So(docCount, ShouldEqual, uint64(20))
		So(index.IsReady(), ShouldBeTrue)
	})

	Convey("Test actualizer", t, func() {
		fakeTS := int64(12345)
		index.indexActualizedTS = fakeTS
		Convey("Test deletion", func() {
			dataBase.EXPECT().FetchTriggersToReindex(fakeTS).Return(triggerIDs[18:20], nil)
			dataBase.EXPECT().GetTriggerChecks(triggerIDs[18:20]).Return([]*moira.TriggerCheck{nil, nil}, nil)

			err := index.actualizeIndex()
			So(err, ShouldBeNil)
			docCount, _ := index.index.DocCount()
			So(docCount, ShouldEqual, uint64(18))
		})

		Convey("Test addition", func() {
			dataBase.EXPECT().FetchTriggersToReindex(fakeTS).Return(triggerIDs[18:20], nil)
			dataBase.EXPECT().GetTriggerChecks(triggerIDs[18:20]).Return(triggersPointers[18:20], nil)

			err := index.actualizeIndex()
			So(err, ShouldBeNil)
			docCount, _ := index.index.DocCount()
			So(docCount, ShouldEqual, uint64(20))
		})

		Convey("Test reindexing old ones", func() {
			dataBase.EXPECT().FetchTriggersToReindex(fakeTS).Return(triggerIDs[10:12], nil)
			dataBase.EXPECT().GetTriggerChecks(triggerIDs[10:12]).Return(triggersPointers[10:12], nil)

			err := index.actualizeIndex()
			So(err, ShouldBeNil)
			docCount, _ := index.index.DocCount()
			So(docCount, ShouldEqual, uint64(20))
		})
	})

}
