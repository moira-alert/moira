package filter

import (
	"fmt"
	"strconv"

	"github.com/moira-alert/moira"
)

// ParseMetric parses metric from string
// supported format: "<metricString> <valueFloat64> <timestampInt64>"
func ParseMetric(input []byte) (string, map[string]string, float64, int64, error) {
	if !isPrintableASCII(input) {
		return "", nil, 0, 0, fmt.Errorf("non-ascii or non-printable chars in metric name: '%s'", input)
	}

	var metricsBytes, valueBytes, timestampBytes []byte
	inputScanner := moira.SplitBytes(input, ' ')
	if !inputScanner.HasNext() {
		return "", nil, 0, 0, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	metricsBytes = inputScanner.Next()
	if !inputScanner.HasNext() {
		return "", nil, 0, 0, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	valueBytes = inputScanner.Next()
	if !inputScanner.HasNext() {
		return "", nil, 0, 0, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	timestampBytes = inputScanner.Next()
	if inputScanner.HasNext() {
		return "", nil, 0, 0, fmt.Errorf("too many space-separated items: '%s'", input)
	}

	metric, labels, err := parseMetric(metricsBytes)
	if err != nil {
		return "", nil, 0, 0, fmt.Errorf("cannot parse metric: '%s' (%s)", input, err)
	}

	value, err := parseFloat(valueBytes)
	if err != nil {
		return "", nil, 0, 0, fmt.Errorf("cannot parse value: '%s' (%s)", input, err)
	}

	timestamp, err := parseFloat(timestampBytes)
	if err != nil {
		return "", nil, 0, 0, fmt.Errorf("cannot parse timestamp: '%s' (%s)", input, err)
	}

	return metric, labels, value, int64(timestamp), nil
}

func parseMetric(metricBytes []byte) (string, map[string]string, error) {
	metricBytesScanner := moira.SplitBytes(metricBytes, ';')
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
		labelBytesScanner := moira.SplitBytes(labelBytes, '=')

		var labelNameBytes, labelValueBytes []byte
		if !labelBytesScanner.HasNext() {
			return "", nil, fmt.Errorf("too few equal-separated items: '%s'", labelBytes)
		}
		labelNameBytes = labelBytesScanner.Next()
		if !labelBytesScanner.HasNext() {
			return "", nil, fmt.Errorf("too few equal-separated items: '%s'", labelBytes)
		}
		labelValueBytes = labelBytesScanner.Next()
		if labelBytesScanner.HasNext() {
			return "", nil, fmt.Errorf("too many equal-separated items: '%s'", labelBytes)
		}
		if len(labelNameBytes) == 0 {
			return "", nil, fmt.Errorf("empty label name: '%s'", labelBytes)
		}
		labelName := moira.UnsafeBytesToString(labelNameBytes)
		labelValue := moira.UnsafeBytesToString(labelValueBytes)
		labels[labelName] = labelValue
	}
	labels["name"] = name
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
