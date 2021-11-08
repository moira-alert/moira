package filter

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/filter"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

func BenchmarkProcessIncomingMetric(b *testing.B) {
	patternsTxt, err := os.Open("patterns.txt")
	if err != nil {
		b.Errorf(err.Error())
	}
	patterns := make([]string, 0)
	patternsReader := bufio.NewReader(patternsTxt)
	for {
		pattern, _, err1 := patternsReader.ReadLine()
		if err1 != nil {
			break
		}
		patterns = append(patterns, string(pattern))
	}
	if err != nil && err != io.EOF {
		b.Errorf("Error reading patterns: %s", err.Error())
	}

	filterMetrics := metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())

	mockCtrl := gomock.NewController(b)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	database.EXPECT().AllowStale().AnyTimes().Return(database)
	logger, _ := logging.GetLogger("Benchmark")

	database.EXPECT().GetPatterns().Return(patterns, nil)
	patternsStorage, err := filter.NewPatternStorage(database, filterMetrics, logger)
	if err != nil {
		b.Errorf("Can not create new cache storage %s", err)
	}
	testMetricsLines := generateMetrics(patternsStorage, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		patternsStorage.ProcessIncomingMetric([]byte(testMetricsLines[i]))
	}
}

func generateMetrics(patternStorage *filter.PatternStorage, count int) []string {
	result := make([]string, 0, count)
	timestamp := time.Now()
	patternTree := patternStorage.PatternIndex.Load().(*filter.PatternIndex).Root
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
		value := rand.Float32()
		ts := fmt.Sprintf("%d", timestamp.Unix())
		v := fmt.Sprintf("%f", value)
		path := strings.Join(parts, ".")
		metric := strings.Join([]string{path, v, ts}, " ")
		result = append(result, metric)
		timestamp = timestamp.Add(time.Microsecond)
	}

	return result
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
