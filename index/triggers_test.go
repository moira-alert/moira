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
	defer mockCtrl.Finish()

	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Benchmark")

	searchIndex := NewSearchIndex(logger, database)

	var indexSizes = []int64{10, 50, 100, 500, 1000, 5000, 10000}

	var numberOfTriggers = 10000

	triggersPointers := make([]*moira.TriggerCheck, numberOfTriggers)
	for i := range triggersPointers {
		description := randStringBytes(500)
		triggersPointers[i] = &moira.TriggerCheck{
			Trigger: moira.Trigger{
				ID:   uuid.NewV4().String(),
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
	indexFake := true

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N && i < len(indexSizes); i++ {
		size := indexSizes[i]
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Start", batchSize, size, indexFake)
		newTriggerPointers := triggersPointers[:size]
		fillIndexWithTriggers(searchIndex, newTriggerPointers, batchSize, indexFake)
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Finish", batchSize, size, indexFake)
	}
}

func BenchmarkIndex_FillWithSeveralBatchSize(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Benchmark")

	searchIndex := NewSearchIndex(logger, database)

	var batchSizes = []int{10, 50, 100, 250, 500, 750, 1000}

	var numberOfTriggers = 1000

	triggersPointers := make([]*moira.TriggerCheck, numberOfTriggers)
	for i := range triggersPointers {
		description := randStringBytes(500)
		triggersPointers[i] = &moira.TriggerCheck{
			Trigger: moira.Trigger{
				ID:   uuid.NewV4().String(),
				Name: randStringBytes(100),
				Desc: &description,
				Tags: []string{randStringBytes(5), randStringBytes(3)},
			},
			LastCheck: moira.CheckData{
				Score: rand.Int63n(1000),
			},
		}
	}

	indexFake := true

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N && i < len(batchSizes); i++ {
		size := batchSizes[i]
		batchSize := size
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Start", batchSize, size, indexFake)
		newTriggerPointers := triggersPointers[:size]
		fillIndexWithTriggers(searchIndex, newTriggerPointers, batchSize, indexFake)
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Finish", batchSize, size, indexFake)
	}

	indexFake = false
	for i := 0; i < b.N && i < len(batchSizes); i++ {
		size := batchSizes[i]
		batchSize := size
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Start", batchSize, size, indexFake)
		newTriggerPointers := triggersPointers[:size]
		fillIndexWithTriggers(searchIndex, newTriggerPointers, batchSize, indexFake)
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Finish", batchSize, size, indexFake)
	}
}

func fillIndexWithTriggers(index *Index, triggerChecksToIndex []*moira.TriggerCheck, batchSize int, indexFakeTrigger bool) {
	index.destroyIndex()
	index.createIndex()

	chunkedTriggersToIndex := chunkTriggerChecks(triggerChecksToIndex, batchSize)
	if indexFakeTrigger {
		index.indexTriggerCheck(fakeTriggerToIndex)
		defer index.index.Delete(fakeTriggerToIndex.ID)
	}

	for _, slice := range chunkedTriggersToIndex {
		index.addBatchOfTriggerChecks(slice)
	}
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
