package bleve

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

func BenchmarkFillIndexFullBatch(b *testing.B) {
	type config struct {
		triggersSize int
		batchSize    int
	}

	testCases := []config{
		{50, 1000},
		{100, 1000},
		{250, 1000},
		{500, 1000},
		{1000, 1000},
		{5000, 1000},
		{10000, 1000},
		{50000, 1000},
		{100, 10},
		{500, 50},
		{1000, 100},
		{2500, 250},
		{5000, 500},
	}

	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("BenchmarkFillIndexFullBatch_Size:%d_Batch:%d", testCase.triggersSize, testCase.batchSize),
			func(b *testing.B) {
				triggersSize := testCase.triggersSize
				batchSize := testCase.batchSize

				triggersPointers := generateTriggerChecks(triggersSize)
				triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

				b.ResetTimer()
				b.ReportAllocs()

				for n := 0; n < b.N; n++ {
					newIndex, _ := CreateTriggerIndex(triggerMapping)
					fillIndexWithTriggers(newIndex, triggersPointers, batchSize)
				}
			})
	}
}

func BenchmarkFillAlwaysEmptyIndex(b *testing.B) {
	testCases := []int{50, 100, 300, 500, 1000, 3000, 5000, 10000}

	for _, batchSize := range testCases {
		b.Run(fmt.Sprintf("BenchmarkFillAlwaysEmptyIndex_Batch:%d", batchSize),
			func(b *testing.B) {
				triggersPointers := generateTriggerChecks(batchSize)
				triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

				b.ResetTimer()
				b.ReportAllocs()

				for n := 0; n < b.N; n++ {
					newIndex, _ := CreateTriggerIndex(triggerMapping)
					fillIndexWithTriggers(newIndex, triggersPointers, batchSize)
				}
			})
	}
}

func BenchmarkFillAlreadyFilledIndex(b *testing.B) {
	testCases := []int{50, 100, 300, 500, 1000}

	for _, batchSize := range testCases {
		b.Run(fmt.Sprintf("BenchmarkFillAlreadyFilledIndex_Batch:%d", batchSize),
			func(b *testing.B) {
				triggersPointers := generateTriggerChecks(batchSize)
				triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

				b.ResetTimer()
				b.ReportAllocs()

				for n := 0; n < b.N; n++ {
					newIndex, _ := CreateTriggerIndex(triggerMapping)
					fillIndexWithTriggers(newIndex, triggersPointers, batchSize)
				}
			})
	}
}

func BenchmarkEmptySearch(b *testing.B) {
	type config struct {
		triggersCount int
		pageSize      int64
	}

	testCases := []config{
		{50, 10},
		{100, 10},
		{250, 10},
		{500, 10},
		{1000, 10},
		{5000, 10},
		{10000, 10},
		{50000, 10},
	}

	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("BenchmarkEmptySearch_TriggersCount:%d_PageSize_%d", testCase.triggersCount, testCase.pageSize),
			func(b *testing.B) {
				triggersPointers := generateTriggerChecks(testCase.triggersCount)
				triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})
				newIndex, _ := CreateTriggerIndex(triggerMapping)
				fillIndexWithTriggers(newIndex, triggersPointers, testCase.triggersCount)

				b.ResetTimer()
				b.ReportAllocs()

				for n := 0; n < b.N; n++ {
					newIndex.Search(make([]string, 0), "", false, 0, testCase.pageSize)
				}
			})
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
