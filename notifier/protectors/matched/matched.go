package matched

import (
	"fmt"
	"strconv"

	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	database moira.Database
	logger   moira.Logger
	matchedK int64
}

// Init configures protector
func (protector *Protector) Init(protectorSettings map[string]string, database moira.Database, logger moira.Logger) error {
	var err error
	protector.database = database
	protector.logger = logger
	protector.matchedK, err = strconv.ParseInt(protectorSettings["k"], 10, 64)
	if err != nil {
		return fmt.Errorf("can not read sentinel matched k from config: %s", err.Error())
	}
	return nil
}

// GetInitialValues returns initial values for protector
func (protector *Protector) GetInitialValues() []int64 {
	return []int64{0, 0}
}

// GetCurrentValues returns current values based on previously taken values
func (protector *Protector) GetCurrentValues(oldValues []int64) ([]int64, error) {
	newValues := make([]int64, len(oldValues))
	newCount, err := protector.database.GetMatchedMetricsUpdatesCount()
	if err != nil {
		return oldValues, err
	}
	newDelta := newCount - oldValues[0]
	newValues[0] = newCount
	newValues[1] = newDelta
	return newValues, nil
}

// IsStateDegraded returns true if state is degraded
func (protector *Protector) IsStateDegraded(oldValues []int64, currentValues []int64) bool {
	degraded := currentValues[1] < (oldValues[1] * protector.matchedK)
	if degraded {
		protector.logger.Infof(
			"Matched state degraded. Old value: %d, current value: %d",
			oldValues[1], currentValues[1])
	}
	return degraded
}
