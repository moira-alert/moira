package checker

import (
	"fmt"
	"sort"
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

// ErrTriggerHasEmptyTargets used if additional trigger target has not metrics data after fetch from source
type ErrTriggerHasEmptyTargets struct {
	targets []string
}

// ErrTriggerHasEmptyTargets implementation with error message
func (err ErrTriggerHasEmptyTargets) Error() string {
	return fmt.Sprintf("target t%v has no metrics", strings.Join(err.targets, ", "))
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

	builder.WriteString("Unexpected to have some targets with only one metric.\n")
	builder.WriteString("Expected targets with only one metric:")
	expectedArray := make([]string, 0, len(err.expected))
	for targetName := range err.expected {
		expectedArray = append(expectedArray, targetName)
	}
	sort.Strings(expectedArray)
	for i, targetName := range expectedArray {
		builder.WriteString(targetName)
		if len(expectedArray) > i+1 {
			builder.WriteString(", ")
		}
	}
	builder.WriteRune('\n')

	builder.WriteString("Actual targets with only one pattern:")
	actualArray := make([]string, 0, len(err.actual))
	for targetName := range err.actual {
		actualArray = append(actualArray, targetName)
	}
	sort.Strings(expectedArray)
	for _, targetName := range actualArray {
		builder.WriteRune('\n')
		builder.WriteRune('\t')
		builder.WriteString(targetName)
		builder.WriteString(" — ")
		builder.WriteString(err.actual[targetName])
	}

	return builder.String()
}
