package filter

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira/filter"
)

func BenchmarkProcessIncomingTaggedMetric(b *testing.B) {
	taggedPatterns, err := loadPatterns("tagged_patterns.txt")
	if err != nil {
		b.Errorf(err.Error())
	}

	patternsStorage, err := createPatternsStorage(taggedPatterns, b)
	if err != nil {
		b.Errorf("Can not create new patterns storage %s", err)
	}

	testMetricsLines := generateTaggedMetrics(taggedPatterns, b.N)

	runBenchmark(b, patternsStorage, testMetricsLines)
}

func generateTaggedMetrics(taggedPatterns *[]string, count int) *[]string {
	result := make([]string, 0, 3*count)
	timestamp := time.Now()

	for i := 0; i < count; i++ {
		taggedPattern := (*taggedPatterns)[rand.Intn(len(*taggedPatterns))]
		tagSpecs, _ := filter.ParseSeriesByTag(taggedPattern)

		matchedMetric := generateMatchedMetric(tagSpecs, timestamp.Add(time.Microsecond))
		partiallyMatchedMetric := generatePartiallyMatchedMetric(tagSpecs, timestamp.Add(time.Microsecond))
		notMatchedMetric := generateNotMatchedMetric(tagSpecs, timestamp.Add(time.Microsecond))

		result = append(result, matchedMetric, partiallyMatchedMetric, notMatchedMetric)
		timestamp = timestamp.Add(time.Microsecond)
	}

	return &result
}

func generateMatchedMetric(tagSpecs []filter.TagSpec, timestamp time.Time) string {
	pathParts := make([]string, 0)
	for _, tagSpec := range tagSpecs {
		tagValues := strings.Split(tagSpec.Value, "|")
		tagValue := tagValues[rand.Intn(len(tagValues))]
		pathParts = addTag(pathParts, tagSpec, tagValue)
	}
	metricPath := strings.Join(pathParts, ";")
	return generateMetricLineByPath(metricPath, timestamp)
}

func generatePartiallyMatchedMetric(tagSpecs []filter.TagSpec, timestamp time.Time) string {
	// there will be only one matched tag
	matchedTag := tagSpecs[rand.Intn(len(tagSpecs))]
	randomTagValueLength := 5

	pathParts := make([]string, 0)
	for _, tagSpec := range tagSpecs {
		var tagValue string
		if tagSpec.Name == matchedTag.Name {
			tagValues := strings.Split(tagSpec.Value, "|")
			tagValue = tagValues[rand.Intn(len(tagValues))]
		} else {
			tagValue = RandStringBytes(randomTagValueLength)
		}
		pathParts = addTag(pathParts, tagSpec, tagValue)
	}
	metricPath := strings.Join(pathParts, ";")
	return generateMetricLineByPath(metricPath, timestamp)
}

func generateNotMatchedMetric(tagSpecs []filter.TagSpec, timestamp time.Time) string {
	randomTagValueLength := 5
	pathParts := make([]string, 0)
	for _, tagSpec := range tagSpecs {
		tagValue := RandStringBytes(randomTagValueLength)
		pathParts = addTag(pathParts, tagSpec, tagValue)
	}
	metricPath := strings.Join(pathParts, ";")
	return generateMetricLineByPath(metricPath, timestamp)
}

func addTag(pathParts []string, tagSpec filter.TagSpec, tagValue string) []string {
	if tagSpec.Name == "name" {
		// name tag should be prepended
		return append([]string{tagValue}, pathParts...)
	}
	return append(pathParts, fmt.Sprintf("%s=%s", tagSpec.Name, tagValue))
}
