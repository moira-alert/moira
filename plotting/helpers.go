package plotting

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/moira-alert/go-chart"
)

// sortedByLen represents string array to be sorted by length.
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

// sanitizeLabelName shortens label names to max length.
func sanitizeLabelName(label string, maxLabelLength int) string {
	labelLength := utf8.RuneCountInString(label)
	if labelLength > maxLabelLength {
		label = string([]rune(label)[:maxLabelLength-3])
		label += "..."
	}
	return label
}

// percentsOfRange results expected percents of range by given min and max values.
func percentsOfRange(min, max, percent float64) float64 {
	delta := math.Abs(max - min)
	return percent * (delta / 100)
}

// getTimeValueFormatter returns a time formatter with a given format and timezone.
func getTimeValueFormatter(location *time.Location, format string) chart.ValueFormatter {
	return func(v interface{}) string {
		storage := &locationStorage{location: location}
		return storage.formatTimeWithLocation(v, format)
	}
}

// locationStorage is a container to store
// timezone and provide time value formatter.
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
// for values on yaxis and resolved maximal formatted value length.
func getYAxisValuesFormatter(limits plotLimits) (func(v interface{}) string, int) {
	var formatter func(v interface{}) string
	deltaLimits := int64(limits.highest) - int64(limits.lowest)
	if deltaLimits > 10 { //nolint
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

// floatToHumanizedValueFormatter converts floats into humanized strings on y axis of plot.
func floatToHumanizedValueFormatter(v interface{}) string {
	if typed, isTyped := v.(float64); isTyped {
		if math.Abs(typed) < 1000 { //nolint
			return fmt.Sprintf("%.f", typed)
		}
		humanized, postfix := humanize.ComputeSI(typed)
		return fmt.Sprintf("%.2f %s", humanized, strings.ToUpper(postfix))
	}
	return ""
}
