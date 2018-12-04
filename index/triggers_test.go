package index

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

type config struct {
	triggersSize     int
	batchSize        int
	indexFakeTrigger bool
}

var testCases = []config{
	{50, 1000, true},
	{100, 1000, true},
	{250, 1000, true},
	{500, 1000, true},
	{1000, 1000, true},
	{5000, 1000, true},
	{10000, 1000, true},
	{100, 10, true},
	{500, 50, true},
	{1000, 100, true},
	{2500, 250, true},
	{5000, 500, true},
	{10, 1000, false},
	{50, 1000, false},
	{100, 1000, false},
	{250, 1000, false},
	{500, 1000, false},
	{1000, 1000, false},
	{100, 10, false},
	{500, 50, false},
	{1000, 100, false},
	{2500, 250, false},
	{5000, 500, false},
}

func BenchmarkFillIndex(b *testing.B) {
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("BenchmarkFillIndex_Size:%d_Batch:%d_IndexFakeTrigger:%t", testCase.triggersSize, testCase.batchSize, testCase.indexFakeTrigger),
			func(b *testing.B) {
				runBenchmark(b, testCase.triggersSize, testCase.batchSize, testCase.indexFakeTrigger)
			})
	}
}

func runBenchmark(b *testing.B, triggersSize int, batchSize int, indexFakeTrigger bool) {
	logger, _ := logging.GetLogger("Benchmark")
	database := redis.NewDatabase(logger, redis.Config{})

	triggersPointers := generateTriggerChecks(triggersSize)

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		searchIndex := NewSearchIndex(logger, database)
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Start", batchSize, triggersSize, indexFakeTrigger)
		fillIndexWithTriggers(searchIndex, triggersPointers, batchSize, indexFakeTrigger)
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] [IndexFake: %v] Finish", batchSize, triggersSize, indexFakeTrigger)
	}
}

func fillIndexWithTriggers(index *Index, triggerChecksToIndex []*moira.TriggerCheck, batchSize int, indexFakeTrigger bool) {
	chunkedTriggersToIndex := chunkTriggerChecks(triggerChecksToIndex, batchSize)
	if indexFakeTrigger {
		index.indexTriggerCheck(fakeTriggerToIndex)
		defer index.index.Delete(fakeTriggerToIndex.ID)
	}

	for _, slice := range chunkedTriggersToIndex {
		index.addBatchOfTriggerChecks(slice)
	}
}

func generateTriggerChecks(number int) []*moira.TriggerCheck {
	triggersPointers := make([]*moira.TriggerCheck, number)
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
	return triggersPointers
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

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charBytes[rand.Intn(len(charBytes))]
	}
	return string(b)
}

const charBytes = ".,!?-_()+1234567890 abcdefghijklmnopqrstuvwxyz"
