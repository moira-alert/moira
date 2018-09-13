package matched

import (
	"fmt"
	"strconv"

	"github.com/moira-alert/moira"
)

type Sentinel struct {
	database moira.Database
	logger   moira.Logger
	matchedK int64
}

func (sentinel *Sentinel) Init(sentinelSettings map[string]string, database moira.Database, logger moira.Logger) error {
	var err error
	sentinel.database = database
	sentinel.logger = logger
	sentinel.matchedK, err = strconv.ParseInt(sentinelSettings["k"], 10, 64)
	if err != nil {
		return fmt.Errorf("can not read sentinel matched k from config: %s", err.Error())
	}
	return nil
}

func (sentinel *Sentinel) GetInitialValues() []int64 {
	return []int64{0, 0}
}

func (sentinel *Sentinel) GetCurrentValues(oldValues []int64) ([]int64, error) {
	newValues := make([]int64, len(oldValues))
	newCount, err := sentinel.database.GetMatchedMetricsUpdatesCount()
	if err != nil {
		return oldValues, err
	}
	newDelta := newCount - oldValues[0]
	newValues = append(newValues, newCount)
	newValues = append(newValues, newDelta)
	return newValues, nil
}

func (sentinel *Sentinel) IsStateDegraded(oldValues []int64, currentValues []int64) bool {
	degraded := currentValues[1] < (oldValues[1] * sentinel.matchedK)
	if degraded {
		sentinel.logger.Infof(
			"Matched state degraded. Old value: %d, current value: %d",
			oldValues[1], currentValues[1])
	}
	return degraded
}
