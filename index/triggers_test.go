package index

import (
	"math/rand"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
)

func BenchmarkSize10Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 10, 1000, true)
}

func BenchmarkSize50Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 50, 1000, true)
}

func BenchmarkSize100Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 100, 1000, true)
}

func BenchmarkSize250Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 250, 1000, true)
}

func BenchmarkSize500Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 500, 1000, true)
}

func BenchmarkSize1000Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 1000, 1000, true)
}

func BenchmarkSize5000Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 5000, 1000, true)
}

func BenchmarkSize10000Batch1000WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 10000, 1000, true)
}

func BenchmarkSize100Batch10WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 100, 10, true)
}

func BenchmarkSize500Batch50WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 500, 50, true)
}

func BenchmarkSize1000Batch100WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 1000, 100, true)
}

func BenchmarkSize2500Batch250WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 2500, 250, true)

}

func BenchmarkSize5000Batch500WithFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 5000, 500, true)

}

func BenchmarkSize10Batch1000WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 10, 1000, false)

}

func BenchmarkSize50Batch1000WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 50, 1000, false)
}

func BenchmarkSize100Batch1000WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 100, 1000, false)

}

func BenchmarkSize250Batch1000WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 250, 1000, false)

}

func BenchmarkSize500Batch1000WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 500, 1000, false)

}

func BenchmarkSize1000Batch1000WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 1000, 1000, false)

}

func BenchmarkSize100Batch10WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 100, 10, false)

}

func BenchmarkSize500Batch50WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 500, 50, false)

}

func BenchmarkSize1000Batch100WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 1000, 100, false)

}

func BenchmarkSize2500Batch250WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 2500, 250, false)

}

func BenchmarkSize5000Batch500WithNoFakeTrigger(b *testing.B) {
	benchmarkIndex(b, 5000, 500, false)
}

func benchmarkIndex(b *testing.B, triggersSize int, batchSize int, indexFakeTrigger bool) {
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
