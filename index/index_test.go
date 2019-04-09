package index

import (
	"fmt"
	"testing"

	bleveOriginal "github.com/blevesearch/bleve"
	"github.com/golang/mock/gomock"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/index/bleve"
	"github.com/moira-alert/moira/index/fixtures"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

func TestIndex_CreateAndFill(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("Test create index", t, func(c C) {
		index := NewSearchIndex(logger, dataBase)
		emptyIndex, _ := bleve.CreateTriggerIndex(bleveOriginal.NewIndexMapping())
		c.So(index.triggerIndex, ShouldHaveSameTypeAs, emptyIndex)
	})

	Convey("Test fill index", t, func(c C) {
		index := NewSearchIndex(logger, dataBase)
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err := index.fillIndex()
		c.So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(32))
	})

	Convey("Test add Triggers to index", t, func(c C) {
		index := NewSearchIndex(logger, dataBase)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err := index.writeByBatches(triggerIDs, defaultIndexBatchSize)
		c.So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(32))
	})

	Convey("Test add Triggers to index, batch size is less than number of triggers", t, func(c C) {
		index := NewSearchIndex(logger, dataBase)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs[:20]).Return(triggerChecksPointers[:20], nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs[20:]).Return(triggerChecksPointers[20:], nil)
		err := index.writeByBatches(triggerIDs, 20)
		c.So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(32))
	})

	Convey("Test add Triggers to index where triggers are already presented", t, func(c C) {
		index := NewSearchIndex(logger, dataBase)

		// first time
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err := index.writeByBatches(triggerIDs, defaultIndexBatchSize)
		c.So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(32))

		// second time
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err = index.writeByBatches(triggerIDs, defaultIndexBatchSize)
		c.So(err, ShouldBeNil)
		docCount, _ = index.triggerIndex.GetCount()
		c.So(docCount, ShouldEqual, int64(32))
	})
}

func TestIndex_Start(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	index := NewSearchIndex(logger, dataBase)

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("Test start and stop index", t, func(c C) {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)

		err := index.Start()
		c.So(err, ShouldBeNil)

		err = index.Stop()
		c.So(err, ShouldBeNil)
	})

	Convey("Test second start during index process", t, func(c C) {
		index.inProgress = true
		index.indexed = false
		err := index.Start()
		c.So(err, ShouldBeNil)
	})

	Convey("Test second start", t, func(c C) {
		index.inProgress = false
		index.indexed = true
		err := index.Start()
		c.So(err, ShouldBeNil)
	})
}

func TestIndex_Errors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	index := NewSearchIndex(logger, dataBase)

	Convey("Test Start index error", t, func(c C) {
		dataBase.EXPECT().GetAllTriggerIDs().Return(make([]string, 0), fmt.Errorf("very bad error"))
		err := index.fillIndex()
		c.So(err, ShouldNotBeNil)
	})
}
