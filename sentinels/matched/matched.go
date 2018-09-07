package matched

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gosexy/to"

	"github.com/moira-alert/moira"
)

type Sentinel struct {
	database       moira.Database
	logger         moira.Logger
	matchedEpsilon time.Duration
	matchedK       int64
	ttl            time.Duration
}

func (sentinel *Sentinel) Init(sentinelSettings map[string]string, database moira.Database, logger moira.Logger) error {
	var err error
	sentinel.database = database
	sentinel.logger = logger
	sentinel.matchedEpsilon = to.Duration(sentinelSettings["epsilon"])
	if sentinel.matchedEpsilon == time.Duration(0) {
		return fmt.Errorf("can not read sentinel matched epsilon from config")
	}
	sentinel.matchedK, err = strconv.ParseInt(sentinelSettings["k"], 10, 64)
	if err != nil {
		return fmt.Errorf("can not read sentinel matched k from config: %s", err.Error())
	}
	sentinel.ttl = to.Duration(sentinelSettings["ttl"])
	if sentinel.ttl == time.Duration(0) {
		return fmt.Errorf("can not read sentinel ttl from config")
	}
	return nil
}

func (sentinel *Sentinel) GetInitialValues() []int64 {
	return []int64{0,0}
}

func (sentinel *Sentinel) GetCurrentValues(oldValues []int64) ([]int64, error) {
	newValues := make([]int64,len(oldValues))
	newCount, err := sentinel.database.GetMatchedMetricsUpdatesCount()
	if err != nil {
		sentinel.logger.Warningf("Can not get current value. Using previous: %d",
			oldValues[1])
		return oldValues, err
	}
	newDelta := newCount - oldValues[0]
	newValues = append(newValues, newCount)
	newValues = append(newValues, newDelta)
	return newValues, nil
}

func (sentinel *Sentinel) IsStateDegraded(oldValues []int64, currentValues []int64) bool {
	degraded := oldValues[1] > (currentValues[1] * sentinel.matchedK)
	if degraded {
		sentinel.logger.Infof(
			"Matched state degraded. Old value: %d, current value: %d",
			oldValues[1], currentValues[1])
	}
	return degraded
}
