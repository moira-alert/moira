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

func TestIndex_actualize(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	index := NewSearchIndex(logger, dataBase)
	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, fill index", t, func(c C) {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs[:20], nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs[:20]).Return(triggerChecksPointers[:20], nil)

		err := index.fillIndex()
		index.indexed = true
		c.So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(20))
	})

	Convey("Test actualizer", t, func(c C) {
		fakeTS := int64(12345)
		index.indexActualizedTS = fakeTS
		Convey("Test deletion", t, func(c C) {
			dataBase.EXPECT().FetchTriggersToReindex(fakeTS).Return(triggerIDs[18:20], nil)
			dataBase.EXPECT().GetTriggerChecks(triggerIDs[18:20]).Return([]*moira.TriggerCheck{nil, nil}, nil)

			err := index.actualizeIndex()
			c.So(err, ShouldBeNil)
			docCount, _ := index.triggerIndex.GetCount()
			c.So(docCount, ShouldEqual, int64(18))
		})

		Convey("Test addition", t, func(c C) {
			dataBase.EXPECT().FetchTriggersToReindex(fakeTS).Return(triggerIDs[18:20], nil)
			dataBase.EXPECT().GetTriggerChecks(triggerIDs[18:20]).Return(triggerChecksPointers[18:20], nil)

			err := index.actualizeIndex()
			c.So(err, ShouldBeNil)
			docCount, _ := index.triggerIndex.GetCount()
			c.So(docCount, ShouldEqual, int64(20))
		})

		Convey("Test reindexing old ones", t, func(c C) {
			dataBase.EXPECT().FetchTriggersToReindex(fakeTS).Return(triggerIDs[10:12], nil)
			dataBase.EXPECT().GetTriggerChecks(triggerIDs[10:12]).Return(triggerChecksPointers[10:12], nil)

			err := index.actualizeIndex()
			c.So(err, ShouldBeNil)
			docCount, _ := index.triggerIndex.GetCount()
			c.So(docCount, ShouldEqual, int64(20))
		})
	})
}
