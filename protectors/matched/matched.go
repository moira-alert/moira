package matched

import (
	"time"

	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	database moira.Database
	logger   moira.Logger
	matchedK int
}

// NewProtector returns new protector
func NewProtector(protectorConfig moira.ProtectorConfig, database moira.Database, logger moira.Logger) (*Protector, error) {
	return &Protector{
		database: database,
		logger:   logger,
		matchedK: protectorConfig.Threshold,
	}, nil
}

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() <-chan moira.ProtectorData {
	ch := make(chan moira.ProtectorData)
	go func() {
		protectorSamples := make([]moira.ProtectorSample, 0)
		for {
			matched, _ := protector.database.GetMatchedMetricsUpdatesCount()
			protectorSamples = append(protectorSamples, moira.ProtectorSample{
				Value: float64(matched),
			})
			if len(protectorSamples) == 2 {
				protectorData := moira.ProtectorData{
					Samples: protectorSamples,
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
	current := protectorData.Samples[1].Value
	previous := protectorData.Samples[0].Value
	degraded := current < previous * float64(protector.matchedK)
	if degraded {
		protector.logger.Infof(
			"Matched state degraded. Old value: %.2f, current value: %.2f",
			current, previous)
	}
	_, err := protector.database.GetNotifierState()
	if err != nil {
		protector.logger.Warningf("Can not get notifier state: %s", err.Error())
	}
	return nil
}
