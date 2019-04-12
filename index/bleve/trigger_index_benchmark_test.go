package bleve

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
	"github.com/op/go-logging"
)

type config struct {
	triggersSize int
	batchSize    int
}

var testCases = []config{
	{50, 1000},
	{100, 1000},
	{250, 1000},
	{500, 1000},
	{1000, 1000},
	{5000, 1000},
	{10000, 1000},
	{100, 10},
	{500, 50},
	{1000, 100},
	{2500, 250},
	{5000, 500},
}

func BenchmarkFillIndex(b *testing.B) {
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("BenchmarkFillIndex_Size:%d_Batch:%d", testCase.triggersSize, testCase.batchSize),
			func(b *testing.B) {
				runBenchmark(b, testCase.triggersSize, testCase.batchSize)
			})
	}
}

func runBenchmark(b *testing.B, triggersSize int, batchSize int) {
	logger, _ := logging.GetLogger("Benchmark")
	triggersPointers := generateTriggerChecks(triggersSize)
	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		newIndex, _ := CreateTriggerIndex(triggerMapping)
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] Start", batchSize, triggersSize)
		fillIndexWithTriggers(newIndex, triggersPointers, batchSize)
		logger.Infof("[Benchmark] [BatchSize: %d] [Triggers: %d] Finish", batchSize, triggersSize)
	}
}

func fillIndexWithTriggers(index *TriggerIndex, triggerChecksToIndex []*moira.TriggerCheck, batchSize int) {
	chunkedTriggersToIndex := chunkTriggerChecks(triggerChecksToIndex, batchSize)
	for _, slice := range chunkedTriggersToIndex {
		index.Write(slice)
	}
}

func generateTriggerChecks(number int) []*moira.TriggerCheck {
	triggersPointers := make([]*moira.TriggerCheck, number)
	for i := range triggersPointers {
		description := randStringBytes(500)
		triggersPointers[i] = &moira.TriggerCheck{
			Trigger: moira.Trigger{
				ID:   uuid.Must(uuid.NewV4()).String(),
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
