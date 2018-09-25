package random

import (
	"time"

	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	database moira.Database
	logger   moira.Logger
	capacity int
}

// NewProtector returns new protector
func NewProtector(protectorConfig moira.ProtectorConfig, database moira.Database, logger moira.Logger) (*Protector, error) {
	capacity := protectorConfig.PointsToFetch
	return &Protector{
		database: database,
		logger:   logger,
		capacity: capacity,
	}, nil
}

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() <-chan moira.ProtectorData {
	ch := make(chan moira.ProtectorData)
	go func() {
		protectorSamples := make([]moira.ProtectorSample, 0)
		for {
			metrics := make([]string, 0)
			randomPatterns, _ := protector.database.GetRandomPatterns(protector.capacity)
			for patternInd := range randomPatterns {
				randomMetric, _ := protector.database.GetPatternRandomMetrics(randomPatterns[patternInd], 1)
				metrics = append(metrics, randomMetric[0])
			}
			metricValues, _ := protector.database.GetMetricsValues(metrics, 0, 1)
			for _, metricValue := range metricValues {
				protectorSamples = append(protectorSamples, moira.ProtectorSample{
					Value: metricValue[0].Value,
				})
			}
			if len(protectorSamples) == 2 {
				protectorData := moira.ProtectorData{
					Samples:   protectorSamples,
					Timestamp: time.Now().UTC().Unix(),
				}
				ch <- protectorData
				protectorSamples = nil
			}
			time.Sleep(time.Second)
		}
	}()
	return ch
}

// Protect performs Nodata protection
func (protector *Protector) Protect(protectorData moira.ProtectorData) error {
	return nil
}
