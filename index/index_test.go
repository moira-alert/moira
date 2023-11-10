package index

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira/metrics"

	bleveOriginal "github.com/blevesearch/bleve/v2"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/index/bleve"
	"github.com/moira-alert/moira/index/fixtures"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

func TestGetTriggerChecksWithRetries(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())
	randomBatch := []string{"123", "123"}
	expectedData := []*moira.TriggerCheck{{Throttling: 123}}
	expectedError := fmt.Errorf("random error")

	Convey("Success get data from database with first try", t, func() {
		dataBase.EXPECT().GetTriggerChecks(randomBatch).Return(expectedData, nil)
		actualData, actualError := index.getTriggerChecksWithRetries(randomBatch)
		So(actualData, ShouldResemble, expectedData)
		So(actualError, ShouldBeEmpty)
	})

	Convey("Success get data from database with third try", t, func() {
		dataBase.EXPECT().GetTriggerChecks(randomBatch).Return(nil, expectedError).Times(2)
		dataBase.EXPECT().GetTriggerChecks(randomBatch).Return(expectedData, nil)
		actualData, actualError := index.getTriggerChecksWithRetries(randomBatch)
		So(actualData, ShouldResemble, expectedData)
		So(actualError, ShouldBeEmpty)
	})

	Convey("Fail get data from database with three tries", t, func() {
		dataBase.EXPECT().GetTriggerChecks(randomBatch).Return(nil, expectedError).Times(3)
		actualData, actualError := index.getTriggerChecksWithRetries(randomBatch)
		So(actualData, ShouldBeEmpty)
		So(actualError.Error(), ShouldResemble, fmt.Sprintf("cannot get trigger checks from DB after 3 tries, last error: %v", expectedError))
	})
}

func TestIndex_CreateAndFill(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("Test create index", t, func() {
		index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())
		emptyIndex, _ := bleve.CreateTriggerIndex(bleveOriginal.NewIndexMapping())
		So(index.triggerIndex, ShouldHaveSameTypeAs, emptyIndex)
	})

	Convey("Test fill index", t, func() {
		index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err := index.fillIndex()
		So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		So(docCount, ShouldEqual, int64(32))
	})

	Convey("Test add Triggers to index", t, func() {
		index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err := index.writeByBatches(triggerIDs, defaultIndexBatchSize)
		So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		So(docCount, ShouldEqual, int64(32))
	})

	Convey("Test add Triggers to index, batch size is less than number of triggers", t, func() {
		const batchSize = 20
		dataBase.EXPECT().GetTriggerChecks(triggerIDs[:batchSize]).Return(triggerChecksPointers[:batchSize], nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs[batchSize:]).Return(triggerChecksPointers[batchSize:], nil)

		index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())
		err := index.writeByBatches(triggerIDs, batchSize)
		So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		So(docCount, ShouldEqual, int64(32))
	})

	Convey("Test check error handling in the handleTriggerBatches", t, func() {
		index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())
		expectedError := fmt.Errorf("test")

		dataBase.EXPECT().GetTriggerChecks(triggerIDs[:20]).Return(triggerChecksPointers[:20], nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs[20:]).Return(triggerChecksPointers[20:], expectedError).Times(3)
		err := index.writeByBatches(triggerIDs, 20)
		So(err, ShouldNotBeNil)
	})

	Convey("Test add Triggers to index where triggers are already presented", t, func() {
		index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())

		// first time
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err := index.writeByBatches(triggerIDs, defaultIndexBatchSize)
		So(err, ShouldBeNil)
		docCount, _ := index.triggerIndex.GetCount()
		So(docCount, ShouldEqual, int64(32))

		// second time
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)
		err = index.writeByBatches(triggerIDs, defaultIndexBatchSize)
		So(err, ShouldBeNil)
		docCount, _ = index.triggerIndex.GetCount()
		So(docCount, ShouldEqual, int64(32))
	})
}

func TestIndex_Start(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("Test start and stop index", t, func() {
		dataBase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		dataBase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggerChecksPointers, nil)

		err := index.Start()
		So(err, ShouldBeNil)

		err = index.Stop()
		So(err, ShouldBeNil)
	})

	Convey("Test second start during index process", t, func() {
		index.inProgress = true
		index.indexed = false
		err := index.Start()
		So(err, ShouldBeNil)
	})

	Convey("Test second start", t, func() {
		index.inProgress = false
		index.indexed = true
		err := index.Start()
		So(err, ShouldBeNil)
	})
}

func TestIndex_Errors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	index := NewSearchIndex(logger, dataBase, metrics.NewDummyRegistry())

	Convey("Test Start index error", t, func() {
		dataBase.EXPECT().GetAllTriggerIDs().Return(make([]string, 0), fmt.Errorf("very bad error"))
		err := index.fillIndex()
		So(err, ShouldNotBeNil)
	})
}
