package index

import (
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

func BenchmarkIndex_CreateAndFill(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Benchmark")
	defer mockCtrl.Finish()

	var numberOfTriggers = 10000

	triggerIDs := make([]string, numberOfTriggers)
	for i := range triggerIDs {
		triggerIDs[i] = uuid.NewV4().String()
	}

	triggersPointers := make([]*moira.TriggerCheck, numberOfTriggers)
	for i := range triggerIDs {
		triggersPointers[i] = &moira.TriggerCheck{
			Trigger: moira.Trigger{
				ID:   triggerIDs[i],
				Name: randStringBytes(100),
				Tags: []string{randStringBytes(5), randStringBytes(3)},
			},
			LastCheck: moira.CheckData{
				Score: rand.Int63n(1000),
			},
		}
	}

	b.ResetTimer()

	batchSize := 1000
	searchIndex := index.NewSearchIndex(logger, database)

	chunkedTriggerIDs := moira.ChunkSlice(triggerIDs, batchSize)
	chunkedTriggerPointers := chunkTriggerChecks(triggersPointers, batchSize)

	database.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)

	for i, slice := range chunkedTriggerIDs {
		database.EXPECT().GetTriggerChecks(slice).Return(chunkedTriggerPointers[i], nil)
	}

	searchIndex.Start()

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
