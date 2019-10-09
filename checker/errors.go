package checker

import (
	"fmt"
	"strings"
)

// ErrTriggerNotExists used if trigger to check does not exists
var ErrTriggerNotExists = fmt.Errorf("trigger does not exists")

// ErrTriggerHasOnlyWildcards used if trigger has only wildcard metrics
type ErrTriggerHasOnlyWildcards struct{}

// ErrTriggerHasOnlyWildcards implementation with constant error message
func (err ErrTriggerHasOnlyWildcards) Error() string {
	return "Trigger never received metrics"
}

// ErrTriggerHasSameMetricNames used if trigger has two metric data with same name
type ErrTriggerHasSameMetricNames struct {
	duplicates map[string][]string
}

// NewErrTriggerHasSameMetricNames is a constructor function for ErrTriggerHasSameMetricNames.
func NewErrTriggerHasSameMetricNames(duplicates map[string][]string) ErrTriggerHasSameMetricNames {
	return ErrTriggerHasSameMetricNames{
		duplicates: duplicates,
	}
}

// ErrTriggerHasSameMetricNames implementation with constant error message
func (err ErrTriggerHasSameMetricNames) Error() string {
	var builder strings.Builder
	builder.WriteString("Targets have metrics with identical name: ")
	for target, duplicates := range err.duplicates {
		builder.WriteString(target)
		builder.WriteRune(':')
		builder.WriteString(strings.Join(duplicates, ", "))
		builder.WriteString("; ")
	}
	return builder.String()
}

// ErrTargetHasNoMetrics used if additional trigger target has not metrics data after fetch from source
type ErrTargetHasNoMetrics struct {
	targetIndex int
}

// ErrTargetHasNoMetrics implementation with constant error message
func (err ErrTargetHasNoMetrics) Error() string {
	return fmt.Sprintf("target t%v has no metrics", err.targetIndex+1)
}

// ErrUnexpectedAloneMetric is an error that fired by checker if alone metrics do not
// match alone metrics specified in trigger.
type ErrUnexpectedAloneMetric struct {
	expected map[string]bool
	actual   map[string]string
}

// NewErrUnexpectedAloneMetric is a constructor function that creates ErrUnexpectedAloneMetric.
func NewErrUnexpectedAloneMetric(expected map[string]bool, actual map[string]string) ErrUnexpectedAloneMetric {
	return ErrUnexpectedAloneMetric{
		expected: expected,
		actual:   actual,
	}
}

// Error is a function that implements error interface.
func (err ErrUnexpectedAloneMetric) Error() string {
	var builder strings.Builder

	builder.WriteString("Unexpected to have some targets with only one pattern.\nExpected targets with only one pattern:\n")
	for targetName := range err.expected {
		builder.WriteString(targetName)
		builder.WriteRune('\n')
	}
	builder.WriteString("Actual targets with only one pattern:\n")
	for targetName, patternName := range err.actual {
		builder.WriteString(targetName)
		builder.WriteRune('-')
		builder.WriteString(patternName)
		builder.WriteRune('\n')
	}

	return builder.String()
}
