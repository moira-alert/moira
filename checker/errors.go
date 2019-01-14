package checker

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// ErrTriggerNotExists used if trigger to check does not exists
var ErrTriggerNotExists = fmt.Errorf("trigger does not exists")

// ErrTriggerHasNoTimeSeries used if trigger has no metrics
type ErrTriggerHasNoTimeSeries struct{}

// ErrTriggerHasNoTimeSeries implementation with constant error message
func (err ErrTriggerHasNoTimeSeries) Error() string {
	return fmt.Sprintf("Trigger has no metrics, check your target")
}

// ErrTriggerHasOnlyWildcards used if trigger has only wildcard metrics
type ErrTriggerHasOnlyWildcards struct{}

// ErrTriggerHasOnlyWildcards implementation with constant error message
func (err ErrTriggerHasOnlyWildcards) Error() string {
	return fmt.Sprintf("Trigger never received metrics")
}

// ErrTriggerHasSameTimeSeriesNames used if trigger has two timeseries with same name
type ErrTriggerHasSameTimeSeriesNames struct {
	names []string
}

// ErrTriggerHasSameTimeSeriesNames implementation with constant error message
func (err ErrTriggerHasSameTimeSeriesNames) Error() string {
	return fmt.Sprintf("Trigger has same timeseries names: %s", strings.Join(err.names, ", "))
}

// ErrTargetHasNoTimeSeries used if additional trigger target has not metrics data after fetch from source
type ErrTargetHasNoTimeSeries struct {
	targetIndex int
}

// ErrTargetHasNoTimeSeries implementation with constant error message
func (err ErrTargetHasNoTimeSeries) Error() string {
	return fmt.Sprintf("target t%v has no timeseries", err.targetIndex+1)
}

// ErrWrongTriggerTargets represents targets with inconsistent number of timeseries
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
	wrongTargets.WriteString(" has more than one timeseries")
	return wrongTargets.String()
}
