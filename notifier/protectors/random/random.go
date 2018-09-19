package random

import (
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	enabled  bool
	database moira.Database
	logger   moira.Logger
	metrics  []string
	capacity int
}

// Init configures protector
func (protector *Protector) Init(protectorSettings map[string]string, database moira.Database, logger moira.Logger) error {
	var err error
	protector.database = database
	protector.logger = logger
	protector.capacity, err = strconv.Atoi(protectorSettings["capacity"])
	if err != nil {
		return fmt.Errorf("can not read capacity from config: %s", err.Error())
	}
	protector.enabled = true
	return nil
}

// IsEnabled returns true if protector is enabled
func (protector *Protector) IsEnabled() bool {
	return protector.enabled
}

// GetInitialValues returns initial protector values
func (protector *Protector) GetInitialValues() ([]float64, error) {
	initialValues := make([]float64, protector.capacity)
	metrics := make([]string, protector.capacity)
	randomPatterns, err := protector.database.GetRandomPatterns(protector.capacity)
	if err != nil {
		return nil, err
	}
	for patternInd := range randomPatterns {
		randomMetric, err := protector.database.GetPatternRandomMetrics(randomPatterns[patternInd], 1)
		if err != nil {
			return nil, err
		}
		metrics[patternInd] = randomMetric[0]
	}
	protector.metrics = metrics
	until := time.Now().Unix()
	from := until - time.Minute.Nanoseconds()
	metricValues, err := protector.database.GetMetricsValues(protector.metrics, from, until)
	for _, metricValueList := range metricValues {
		lastElementInd := len(metricValueList) - 1
		initialValues = append(initialValues, metricValueList[lastElementInd].Value)
	}
	return initialValues, nil
}

// GetCurrentValues returns current values based on previously taken values
func (protector *Protector) GetCurrentValues(oldValues []float64) ([]float64, error) {
	return nil, nil
}

// IsStateDegraded returns true if state is degraded
func (protector *Protector) IsStateDegraded(oldValues []float64, currentValues []float64) bool {
	return false
}
	//degraded :=
	//if degraded {
	//	protector.logger.Infof(
	//		"Matched state degraded. Old value: %.2f, current value: %.2f",
	//		oldValues[1], currentValues[1])
	//}
	//return degraded
//}

