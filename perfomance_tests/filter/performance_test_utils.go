package filter

import (
	"bufio"
	"fmt"
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

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func loadPatterns(filename string) (*[]string, error) {
	patternsFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	patterns := make([]string, 0)
	patternsReader := bufio.NewReader(patternsFile)
	for {
		pattern, err1 := patternsReader.ReadString('\n')
		if err1 != nil {
			break
		}
		patterns = append(patterns, pattern[:len(pattern)-1])
	}
	return &patterns, nil
}

func createPatternsStorage(patterns *[]string, b *testing.B) (*filter.PatternStorage, error) {
	mockCtrl := gomock.NewController(b)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	database.EXPECT().GetPatterns().Return(*patterns, nil)

	filterMetrics := metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())
	logger, _ := logging.GetLogger("Benchmark")
	compatibility := filter.Compatibility{RegexTreatment: filter.LooseStartMatch}
	patternsStorage, err := filter.NewPatternStorage(database, filterMetrics, logger, compatibility)
	if err != nil {
		return nil, err
	}
	return patternsStorage, nil
}

func runBenchmark(b *testing.B, patternsStorage *filter.PatternStorage, testMetricsLines *[]string) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMetricLine := (*testMetricsLines)[rand.Intn(len(*testMetricsLines))]
		patternsStorage.ProcessIncomingMetric([]byte(testMetricLine), time.Hour)
	}
}

func generateMetricLineByPath(metricPath string, timestamp time.Time) string {
	timestampString := fmt.Sprintf("%d", timestamp.Unix())
	randomValueString := fmt.Sprintf("%f", rand.Float32())
	return strings.Join([]string{metricPath, randomValueString, timestampString}, " ")
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
