package checker

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// ErrTriggerNotExists used if trigger to check does not exists
var ErrTriggerNotExists = fmt.Errorf("trigger does not exists")

// ErrTriggerHasNoMetrics used if trigger has no metrics
type ErrTriggerHasNoMetrics struct{}

// ErrTriggerHasNoMetrics implementation with constant error message
func (err ErrTriggerHasNoMetrics) Error() string {
	return fmt.Sprintf("Trigger has no metrics, check your target")
}

// ErrTriggerHasOnlyWildcards used if trigger has only wildcard metrics
type ErrTriggerHasOnlyWildcards struct{}

// ErrTriggerHasOnlyWildcards implementation with constant error message
func (err ErrTriggerHasOnlyWildcards) Error() string {
	return fmt.Sprintf("Trigger never received metrics")
}

// ErrTriggerHasSameMetricNames used if trigger has two metric data with same name
type ErrTriggerHasSameMetricNames struct {
	names []string
}

// ErrTriggerHasSameMetricNames implementation with constant error message
func (err ErrTriggerHasSameMetricNames) Error() string {
	return fmt.Sprintf("Several metrics have an identical name: %s", strings.Join(err.names, ", "))
}

// ErrTargetHasNoMetrics used if additional trigger target has not metrics data after fetch from source
type ErrTargetHasNoMetrics struct {
	targetIndex int
}

// ErrTargetHasNoMetrics implementation with constant error message
func (err ErrTargetHasNoMetrics) Error() string {
	return fmt.Sprintf("target t%v has no metrics", err.targetIndex+1)
}

// ErrWrongTriggerTargets represents targets with inconsistent number of metrics
type ErrWrongTriggerTargets []int

// ErrWrongTriggerTarget implementation for list of invalid targets found
func (err ErrWrongTriggerTargets) Error() string {
	var countType []byte
	if len(err) > 1 {
		countType = []byte("Targets ")
	} else {
		countType = []byte("Target ")
	}
	wrongTargets := bytes.NewBuffer(countType)
	for tarInd, tar := range err {
		wrongTargets.WriteString("t")
		wrongTargets.WriteString(strconv.Itoa(tar))
		if tarInd != len(err)-1 {
			wrongTargets.WriteString(", ")
		}
	}
	wrongTargets.WriteString(" has more than one metric")
	return wrongTargets.String()
}
