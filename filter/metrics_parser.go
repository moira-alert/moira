package filter

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira"
)

// ParsedMetric represents a result of ParseMetric.
type ParsedMetric struct {
	Metric    string
	Name      string
	Labels    map[string]string
	Value     float64
	Timestamp int64
}

// ParseMetric parses metric from string
// supported format: "<metricString> <valueFloat64> <timestampInt64>"
func ParseMetric(input []byte) (*ParsedMetric, error) {
	if !isPrintableASCII(input) {
		return nil, fmt.Errorf("non-ascii or non-printable chars in metric name: '%s'", input)
	}

	var metricBytes, valueBytes, timestampBytes []byte
	inputScanner := moira.NewBytesScanner(input, ' ')
	if !inputScanner.HasNext() {
		return nil, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	metricBytes = inputScanner.Next()
	if !inputScanner.HasNext() {
		return nil, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	valueBytes = inputScanner.Next()
	if !inputScanner.HasNext() {
		return nil, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	timestampBytes = inputScanner.Next()
	if inputScanner.HasNext() {
		return nil, fmt.Errorf("too many space-separated items: '%s'", input)
	}

	name, labels, err := parseNameAndLabels(metricBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse metric: '%s' (%s)", input, err)
	}

	value, err := parseFloat(valueBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse value: '%s' (%s)", input, err)
	}

	timestamp, err := parseFloat(timestampBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse timestamp: '%s' (%s)", input, err)
	}

	parsedMetric := &ParsedMetric{
		restoreMetricStringByNameAndLabels(name, labels),
		name,
		labels,
		value,
		int64(timestamp),
	}
	if timestamp == -1 {
		parsedMetric.Timestamp = time.Now().Unix()
	}
	return parsedMetric, nil
}

func restoreMetricStringByNameAndLabels(name string, labels map[string]string) string {
	var builder strings.Builder
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	builder.WriteString(name)

	for _, key := range keys {
		builder.WriteString(fmt.Sprintf(";%s=%s", key, labels[key]))
	}

	return builder.String()
}

// IsTagged checks that metric is tagged
func (metric ParsedMetric) IsTagged() bool {
	return len(metric.Labels) > 0
}

func parseNameAndLabels(metricBytes []byte) (string, map[string]string, error) {
	metricBytesScanner := moira.NewBytesScanner(metricBytes, ';')
	if !metricBytesScanner.HasNext() {
		return "", nil, fmt.Errorf("too few colon-separated items: '%s'", metricBytes)
	}
	nameBytes := metricBytesScanner.Next()
	if len(nameBytes) == 0 {
		return "", nil, fmt.Errorf("empty metric name: '%s'", metricBytes)
	}
	name := moira.UnsafeBytesToString(nameBytes)
	labels := make(map[string]string)
	for metricBytesScanner.HasNext() {
		labelBytes := metricBytesScanner.Next()
		labelBytesScanner := moira.NewBytesScanner(labelBytes, '=')

		var labelNameBytes, labelValueBytes []byte
		if !labelBytesScanner.HasNext() {
			return "", nil, fmt.Errorf("too few equal-separated items: '%s'", labelBytes)
		}
		labelNameBytes = labelBytesScanner.Next()
		if !labelBytesScanner.HasNext() {
			return "", nil, fmt.Errorf("too few equal-separated items: '%s'", labelBytes)
		}
		labelValueBytes = labelBytesScanner.Next()
		for labelBytesScanner.HasNext() {
			var labelString strings.Builder
			labelString.WriteString("=")
			labelString.Write(labelBytesScanner.Next())
			labelValueBytes = append(labelValueBytes, labelString.String()...)
		}
		if len(labelNameBytes) == 0 {
			return "", nil, fmt.Errorf("empty label name: '%s'", labelBytes)
		}
		labelName := moira.UnsafeBytesToString(labelNameBytes)
		labelValue := moira.UnsafeBytesToString(labelValueBytes)
		labels[labelName] = labelValue
	}
	return name, labels, nil
}

func parseFloat(input []byte) (float64, error) {
	return strconv.ParseFloat(moira.UnsafeBytesToString(input), 64)
}

func isPrintableASCII(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] < 0x20 || b[i] > 0x7E {
			return false
		}
	}

	return true
}
