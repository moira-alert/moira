package filter

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira/filter"
)

func BenchmarkProcessIncomingPlainMetric(b *testing.B) {
	plainPatterns, err := loadPatterns("plain_patterns.txt")
	if err != nil {
		b.Errorf(err.Error())
	}

	patternsStorage, err := createPatternsStorage(plainPatterns, b)
	if err != nil {
		b.Errorf("Can not create new patterns storage %s", err)
	}

	testMetricsLines := generateMetrics(patternsStorage, b.N)

	runBenchmark(b, patternsStorage, testMetricsLines)
}

func generateMetrics(patternStorage *filter.PatternStorage, count int) *[]string {
	result := make([]string, 0, count)
	timestamp := time.Now()
	patternTree := patternStorage.PatternIndex.Load().(*filter.PatternIndex).Tree.Root
	for i := 0; i < count; i++ {
		parts := make([]string, 0, 16)

		node := patternTree.Children[rand.Intn(len(patternTree.Children))]
		matched := rand.Float64() < 0.02
		level := float64(0)
		for {
			part := node.Part
			if len(node.InnerParts) > 0 {
				part = node.InnerParts[rand.Intn(len(node.InnerParts))]
			}
			if !matched && rand.Float64() < 0.2+level {
				part = RandStringBytes(len(part))
			}
			parts = append(parts, strings.Replace(part, "*", "XXXXXXXXX", -1))
			if len(node.Children) == 0 {
				break
			}
			level += 0.7
			node = node.Children[rand.Intn(len(node.Children))]
		}
		matchedMetricPath := strings.Join(parts, ".")
		metric := generateMetricLineByPath(matchedMetricPath, timestamp)
		result = append(result, metric)
		timestamp = timestamp.Add(time.Microsecond)
	}

	return &result
}
