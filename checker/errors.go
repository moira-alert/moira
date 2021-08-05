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

// ErrTriggerHasEmptyTargets used if additional trigger target has not metrics data after fetch from source
type ErrTriggerHasEmptyTargets struct {
	targets []string
}

// ErrTriggerHasEmptyTargets implementation with error message
func (err ErrTriggerHasEmptyTargets) Error() string {
	return fmt.Sprintf("target t%v has no metrics", strings.Join(err.targets, ", "))
}

// ErrNetwork used if network error occurred during fetch
type ErrNetwork struct {
	networkError error
}

// ErrNetwork implementation with error message
func (err ErrNetwork) Error() string {
	return fmt.Sprintf("network error during fetch: %s", err.networkError.Error())
}
