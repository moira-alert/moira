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

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() (<-chan []moira.ProtectorData) {
	ch := make(chan []moira.ProtectorData)
	go func() {
		protectorData := make([]moira.ProtectorData, 0)
		for {
			metrics := make([]string, 0)
			randomPatterns, _ := protector.database.GetRandomPatterns(protector.capacity)
			for patternInd := range randomPatterns {
				randomMetric, _ := protector.database.GetPatternRandomMetrics(randomPatterns[patternInd], 1)
				metrics = append(metrics, randomMetric[0])
			}
			metricValues, _ := protector.database.GetMetricsValues(metrics, 0, 1)
			for _, metricValue := range metricValues {
				protectorData = append(protectorData, moira.ProtectorData{
					Value:float64(metricValue[0].Value),
				})
			}
			if len(protectorData) == 2 {
				ch <- protectorData
				protectorData = nil
			}
			time.Sleep(time.Second)
		}
	}()
	return ch
}

// IsStateDegraded returns true if state is degraded
func (protector *Protector) IsStateDegraded(protectorData []moira.ProtectorData) bool {
	degraded := protectorData[1].Value < protectorData[0].Value
	if degraded {
		protector.logger.Infof(
			"Matched state degraded. Old value: %.2f, current value: %.2f",
			protectorData[0].Value, protectorData[1].Value)
	}
	return degraded
}

