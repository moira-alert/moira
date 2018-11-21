package index

import (
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

func BenchmarkIndex_FillWithDefaultBatchSize(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Benchmark")
	defer mockCtrl.Finish()

	var indexSizes = []int64{10, 50, 100, 500, 1000, 5000, 10000}

	var numberOfTriggers = 10000

	triggerIDs := make([]string, numberOfTriggers)
	for i := range triggerIDs {
		triggerIDs[i] = uuid.NewV4().String()
	}

	triggersPointers := make([]*moira.TriggerCheck, numberOfTriggers)
	for i := range triggerIDs {
		description := randStringBytes(500)
		triggersPointers[i] = &moira.TriggerCheck{
			Trigger: moira.Trigger{
				ID:   triggerIDs[i],
				Name: randStringBytes(100),
				Desc: &description,
				Tags: []string{randStringBytes(5), randStringBytes(3)},
			},
			LastCheck: moira.CheckData{
				Score: rand.Int63n(1000),
			},
		}
	}

	batchSize := defaultIndexBatchSize

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N && i < len(indexSizes); i++ {
		size := indexSizes[i]
		logger.Infof("Index %d triggers", size)
		newSearchIndex := NewSearchIndex(logger, database)
		newTriggerIDs := triggerIDs[:size]
		newTriggerPointers := triggersPointers[:size]
		batchFillIndexWithTriggers(newSearchIndex, database, newTriggerIDs, newTriggerPointers, batchSize)
		logger.Info("Successfully indexed %d triggers", size)
	}
}

func BenchmarkIndex_FillWithSeveralBatchSize(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Benchmark")
	defer mockCtrl.Finish()

	var batchSizes = []int{10, 50, 100, 250, 500, 750, 1000}

	var numberOfTriggers = 1000

	triggerIDs := make([]string, numberOfTriggers)
	for i := range triggerIDs {
		triggerIDs[i] = uuid.NewV4().String()
	}

	triggersPointers := make([]*moira.TriggerCheck, numberOfTriggers)
	for i := range triggerIDs {
		description := randStringBytes(500)
		triggersPointers[i] = &moira.TriggerCheck{
			Trigger: moira.Trigger{
				ID:   triggerIDs[i],
				Name: randStringBytes(100),
				Desc: &description,
				Tags: []string{randStringBytes(5), randStringBytes(3)},
			},
			LastCheck: moira.CheckData{
				Score: rand.Int63n(1000),
			},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N && i < len(batchSizes); i++ {
		size := batchSizes[i]
		batchSize := size
		logger.Infof("Index %d triggers", size)
		newSearchIndex := NewSearchIndex(logger, database)
		newTriggerIDs := triggerIDs[:size]
		newTriggerPointers := triggersPointers[:size]
		batchFillIndexWithTriggers(newSearchIndex, database, newTriggerIDs, newTriggerPointers, batchSize)
		logger.Info("Successfully indexed %d triggers", size)
	}

}

func batchFillIndexWithTriggers(index *Index, database *mock_moira_alert.MockDatabase, triggerIDs []string, triggerPointers []*moira.TriggerCheck, batchSize int) {
	chunkedTriggerIDs := moira.ChunkSlice(triggerIDs, batchSize)
	chunkedTriggerPointers := chunkTriggerChecks(triggerPointers, batchSize)

	for i, slice := range chunkedTriggerIDs {
		database.EXPECT().GetTriggerChecks(slice).Return(chunkedTriggerPointers[i], nil)
	}

	index.addTriggers(triggerIDs, batchSize)
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charBytes[rand.Intn(len(charBytes))]
	}
	return string(b)
}

func chunkTriggerChecks(original []*moira.TriggerCheck, chunkSize int) (divided [][]*moira.TriggerCheck) {
	if chunkSize < 1 {
		return
	}
	for i := 0; i < len(original); i += chunkSize {
		end := i + chunkSize

		if end > len(original) {
			end = len(original)
		}

		divided = append(divided, original[i:end])
	}
	return
}

const charBytes = ".,!?-_()+1234567890 abcdefghijklmnopqrstuvwxyz"
