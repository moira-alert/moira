package plotting

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
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

// int64ToTime returns time.Time from int64
func int64ToTime(timeStamp int64, location *time.Location) time.Time {
	return time.Unix(timeStamp, 0).In(location)
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

// floatToHumanizedValueFormatter converts floats into humanized strings on y axis of plot
func floatToHumanizedValueFormatter(v interface{}) string {
	if typed, isTyped := v.(float64); isTyped {
		if math.Abs(typed) < 1000 {
			return fmt.Sprintf("%.f", typed)
		}
		typed, postfix := humanize.ComputeSI(typed)
		return fmt.Sprintf("%.2f %s", typed, strings.ToUpper(postfix))
	}
	return ""
}

// getYAxisValuesFormatter returns value formatter for values on yaxis
func getYAxisValuesFormatter(limits plotLimits) func(v interface{}) string {
	deltaLimits := int64(limits.highest) - int64(limits.lowest)
	if deltaLimits > 10 {
		return floatToHumanizedValueFormatter
	}
	return chart.FloatValueFormatter
}
