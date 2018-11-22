package index

import (
	"math/rand"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

func BenchmarkFillIndex(b *testing.B) {
	b.Run("Benchmark: Size 10, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 10, 1000, true) })
	b.Run("Benchmark: Size 50, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 50, 1000, true) })
	b.Run("Benchmark: Size 100, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 100, 1000, true) })
	b.Run("Benchmark: Size 250, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 250, 1000, true) })
	b.Run("Benchmark: Size 500, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 500, 1000, true) })
	b.Run("Benchmark: Size 1000, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 1000, 1000, true) })
	b.Run("Benchmark: Size 5000, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 5000, 1000, true) })
	b.Run("Benchmark: Size 10000, Batch 1000, Index fake trigger", func(b *testing.B) { runBenchmark(b, 10000, 1000, true) })
	b.Run("Benchmark: Size 100, Batch 10, Index fake trigger", func(b *testing.B) { runBenchmark(b, 100, 10, true) })
	b.Run("Benchmark: Size 500, Batch 50, Index fake trigger", func(b *testing.B) { runBenchmark(b, 500, 50, true) })
	b.Run("Benchmark: Size 1000, Batch 100, Index fake trigger", func(b *testing.B) { runBenchmark(b, 1000, 100, true) })
	b.Run("Benchmark: Size 2500, Batch 250, Index fake trigger", func(b *testing.B) { runBenchmark(b, 2500, 250, true) })
	b.Run("Benchmark: Size 5000, Batch 500, Index fake trigger", func(b *testing.B) { runBenchmark(b, 5000, 500, true) })
	b.Run("Benchmark: Size 10, Batch 1000, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 10, 1000, false) })
	b.Run("Benchmark: Size 50, Batch 1000, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 50, 1000, false) })
	b.Run("Benchmark: Size 100, Batch 1000, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 100, 1000, false) })
	b.Run("Benchmark: Size 250, Batch 1000, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 250, 1000, false) })
	b.Run("Benchmark: Size 500, Batch 1000, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 500, 1000, false) })
	b.Run("Benchmark: Size 1000, Batch 1000, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 1000, 1000, false) })
	b.Run("Benchmark: Size 100, Batch 10, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 100, 10, false) })
	b.Run("Benchmark: Size 500, Batch 50, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 500, 50, false) })
	b.Run("Benchmark: Size 1000, Batch 100, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 1000, 100, false) })
	b.Run("Benchmark: Size 2500, Batch 250, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 2500, 250, false) })
	b.Run("Benchmark: Size 5000, Batch 500, NO index fake trigger", func(b *testing.B) { runBenchmark(b, 5000, 500, false) })
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
