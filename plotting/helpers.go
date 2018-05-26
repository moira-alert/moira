package plotting

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/wcharczuk/go-chart"
)

// SortedByLen represents string array to be sorted by length
type SortedByLen []string

func (initial SortedByLen) Len() int {
	return len(initial)
}

func (initial SortedByLen) Less(i int, j int) bool {
	return len(initial[i]) < len(initial[j])
}

func (initial SortedByLen) Swap(i int, j int) {
	initial[i], initial[j] = initial[j], initial[i]
}

// Int32ToTime returns time.Time from int32
func Int32ToTime(timeStamp int32) time.Time {
	return time.Unix(int64(timeStamp), 0).UTC()
}

// SanitizeLabelName shortens label names to max length
func SanitizeLabelName(label string, maxLabelLength int) string {
	labelLength := len(label)
	if labelLength > maxLabelLength {
		label = label[:maxLabelLength-3]
		label += "..."
	}
	return label
}

// FloatToHumanizedValueFormatter converts floats into humanized strings on y axis of plot
func FloatToHumanizedValueFormatter(v interface{}) string {
	if typed, isTyped := v.(float64); isTyped {
		if math.Abs(typed) < 1000 {
			return fmt.Sprintf("%.f", typed)
		}
		typed, postfix := humanize.ComputeSI(typed)
		return fmt.Sprintf("%.2f %s", typed, strings.ToUpper(postfix))
	}
	return ""
}

// GetYAxisValuesFormatter returns value formatter for values on yaxis
func GetYAxisValuesFormatter(plotLimits Limits) func(v interface{}) string {
	deltaLimits := int64(plotLimits.Highest) - int64(plotLimits.Lowest)
	if deltaLimits > 10 {
		return FloatToHumanizedValueFormatter
	}
	return chart.FloatValueFormatter
}
