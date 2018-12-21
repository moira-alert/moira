package plotting

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"
)

// sortedByLen represents string array to be sorted by length
type sortedByLen []string

func (initial sortedByLen) Len() int {
	return len(initial)
}

func (initial sortedByLen) Less(i int, j int) bool {
	return len(initial[i]) < len(initial[j])
}

func (initial sortedByLen) Swap(i int, j int) {
	initial[i], initial[j] = initial[j], initial[i]
}

// sanitizeLabelName shortens label names to max length
func sanitizeLabelName(label string, maxLabelLength int) string {
	labelLength := len(label)
	if labelLength > maxLabelLength {
		label = label[:maxLabelLength-3]
		label += "..."
	}
	return label
}

// getTimeValueFormatter returns a time formatter with a given format and timezone
func getTimeValueFormatter(location *time.Location, format string) chart.ValueFormatter {
	return func(v interface{}) string {
		storage := &locationStorage{location: location}
		return storage.formatTimeWithLocation(v, format)
	}
}

// locationStorage is a container to store
// timezone and provide time value formatter
type locationStorage struct {
	location *time.Location
}

// TimeValueFormatterWithFormat is a ValueFormatter for timestamps with a given format.
func (storage locationStorage) formatTimeWithLocation(v interface{}, dateFormat string) string {
	if typed, isTyped := v.(time.Time); isTyped {
		return typed.In(storage.location).Format(dateFormat)
	}
	if typed, isTyped := v.(int64); isTyped {
		return time.Unix(0, typed).In(storage.location).Format(dateFormat)
	}
	if typed, isTyped := v.(float64); isTyped {
		return time.Unix(0, int64(typed)).In(storage.location).Format(dateFormat)
	}
	return ""
}

// getYAxisValuesFormatter returns value formatter
// for values on yaxis and resolved maximal formatted value length
func getYAxisValuesFormatter(limits plotLimits) (func(v interface{}) string, int) {
	var formatter func(v interface{}) string
	deltaLimits := int64(limits.highest) - int64(limits.lowest)
	if deltaLimits > 10 {
		formatter = floatToHumanizedValueFormatter
	} else {
		formatter = chart.FloatValueFormatter
	}
	lowestLen := len(formatter(limits.lowest))
	highestLen := len(formatter(limits.highest))
	if lowestLen > highestLen {
		return formatter, lowestLen
	}
	return formatter, highestLen
}

// floatToHumanizedValueFormatter converts floats into humanized strings on y axis of plot
func floatToHumanizedValueFormatter(v interface{}) string {
	if typed, isTyped := v.(float64); isTyped {
		if math.Abs(typed) < 1000 {
			return fmt.Sprintf("%.f", typed)
		}
		humanized, postfix := humanize.ComputeSI(typed)
		return fmt.Sprintf("%.2f %s", humanized, strings.ToUpper(postfix))
	}
	return ""
}

// toLimitedMetricsData returns MetricData limited by whitelist
func toLimitedMetricsData(metricsData []*types.MetricData, metricsWhitelist []string) []*types.MetricData {
	if len(metricsWhitelist) == 0 {
		return metricsData
	}
	newMetricsData := make([]*types.MetricData, 0, len(metricsWhitelist))
	for _, metricData := range metricsData {
		if isWhiteListedMetric(metricData.Name, metricsWhitelist) {
			newMetricsData = append(newMetricsData, metricData)
		}
	}
	return newMetricsData
}

// isWhiteListedMetric returns true if metric is whitelisted
func isWhiteListedMetric(metricName string, metricsWhitelist []string) bool {
	for _, whiteListed := range metricsWhitelist {
		if whiteListed == metricName {
			return true
		}
	}
	return false
}
