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

func BenchmarkProcessIncomingTaggedMetric(b *testing.B) {
	taggedPatterns, err := loadTaggedPatterns()
	if err != nil {
		b.Errorf(err.Error())
	}
	filterMetrics := metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())

	mockCtrl := gomock.NewController(b)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	database.EXPECT().AllowStale().AnyTimes().Return(database)
	logger, _ := logging.GetLogger("Benchmark")

	database.EXPECT().GetPatterns().Return(taggedPatterns, nil)
	patternsStorage, err := filter.NewPatternStorage(database, filterMetrics, logger)
	if err != nil {
		b.Errorf("Can not create new cache storage %s", err)
	}
	testMetricsLines := generateTaggedMetrics(&taggedPatterns, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		patternsStorage.ProcessIncomingMetric([]byte(testMetricsLines[i]))
	}
}

func loadTaggedPatterns() ([]string, error) {
	taggedPatternsTxt, err := os.Open("tagged_patterns.txt")
	if err != nil {
		return nil, err
	}
	taggedPatterns := make([]string, 0)
	patternsReader := bufio.NewReader(taggedPatternsTxt)
	lastLineWasPrefixed := false
	for {
		pattern, isPrefix, err1 := patternsReader.ReadLine()
		if err1 != nil {
			break
		}
		if lastLineWasPrefixed {
			taggedPatterns[len(taggedPatterns)-1] = fmt.Sprintf(
				"%s%s", taggedPatterns[len(taggedPatterns)-1], pattern,
			)
		} else {
			taggedPatterns = append(taggedPatterns, string(pattern))
		}
		lastLineWasPrefixed = isPrefix
	}
	return taggedPatterns, nil
}
func generateTaggedMetrics(taggedPatterns *[]string, count int) []string {
	result := make([]string, 0, count)
	timestamp := time.Now()

	for i := 0; i < count; i++ {
		taggedPattern := (*taggedPatterns)[rand.Intn(len(*taggedPatterns))]
		tagSpecs, _ := filter.ParseSeriesByTag(taggedPattern)
		matchedMetricPathParts := make([]string, 0)
		for _, tagSpec := range tagSpecs {
			tagValues := strings.Split(tagSpec.Value, "|")
			tagValue := tagValues[rand.Intn(len(tagValues))]
			if tagSpec.Name == "name" {
				matchedMetricPathParts = append([]string{tagValue}, matchedMetricPathParts...)
			} else {
				matchedMetricPathParts = append(matchedMetricPathParts, fmt.Sprintf("%s=%s", tagSpec.Name, tagValue))
			}
		}
		matchedMetricPath := strings.Join(matchedMetricPathParts, ";")
		value := rand.Float32()
		ts := fmt.Sprintf("%d", timestamp.Unix())
		v := fmt.Sprintf("%f", value)
		metric := strings.Join([]string{matchedMetricPath, v, ts}, " ")
		result = append(result, metric)
		timestamp = timestamp.Add(time.Microsecond)
	}

	return result
}
