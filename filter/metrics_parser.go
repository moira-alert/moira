package filter

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/moira-alert/moira"
)

// ParseMetric parses metric from string
// supported format: "<metricString> <valueFloat64> <timestampInt64>"
func ParseMetric(input []byte) (string, float64, int64, error) {
	firstSpaceIndex := bytes.IndexByte(input, ' ')
	if firstSpaceIndex < 1 {
		return "", 0, 0, fmt.Errorf("too few space-separated items: '%s'", input)
	}

	secondSpaceIndex := bytes.IndexByte(input[firstSpaceIndex+1:], ' ')
	if secondSpaceIndex < 1 {
		return "", 0, 0, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	secondSpaceIndex += firstSpaceIndex + 1

	metric := input[:firstSpaceIndex]
	if !isPrintableASCII(metric) {
		return "", 0, 0, fmt.Errorf("non-ascii or non-printable chars in metric name: '%s'", input)
	}

	value, err := parseFloat(input[firstSpaceIndex+1 : secondSpaceIndex])
	if err != nil {
		return "", 0, 0, fmt.Errorf("cannot parse value: '%s' (%s)", input, err)
	}

	timestamp, err := parseFloat(input[secondSpaceIndex+1:])
	if err != nil {
		return "", 0, 0, fmt.Errorf("cannot parse timestamp: '%s' (%s)", input, err)
	}

	return moira.UnsafeBytesToString(metric), value, int64(timestamp), nil
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
