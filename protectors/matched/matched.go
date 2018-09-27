package matched

import (
	"time"

	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	database    moira.Database
	logger      moira.Logger
	inspectOnly bool
	numSamples  int
	retention   time.Duration
	ratio       float64
	throttling  int
}

// NewProtector returns new protector
func NewProtector(database moira.Database, logger moira.Logger,
	inspectOnly bool, numSamples int, sampleRetention time.Duration,
	sampleRatio float64, throttling int) (*Protector, error) {
	return &Protector{
		database:    database,
		logger:      logger,
		inspectOnly: inspectOnly,
		numSamples:  numSamples,
		retention:   sampleRetention,
		ratio:       sampleRatio,
		throttling:  throttling,
	}, nil
}

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() <-chan moira.ProtectorData {
	ch := make(chan moira.ProtectorData)
	go func() {
		protectorSamples := make([]moira.ProtectorSample, 0)
		protectTicker := time.NewTicker(protector.retention)
		for t := range protectTicker.C {
			if len(protectorSamples) == protector.numSamples {
				protectorData := moira.ProtectorData{
					Samples:   protectorSamples,
					Timestamp: t.Unix(),
				}
				ch <- protectorData
				protectorSamples = nil
			}

			matched, _ := protector.database.GetMatchedMetricsUpdatesCount()
			protectorSamples = append(protectorSamples, moira.ProtectorSample{
				Value: float64(matched),
			})
		}
	}()
	return ch
}

// Protect performs Nodata protection
func (protector *Protector) Protect(protectorData moira.ProtectorData) error {
	var degraded bool
	deltas := make([]float64, len(protectorData.Samples)-1)
	for sampleInd := range protectorData.Samples {
		if sampleInd > 0 {
			delta := protectorData.Samples[sampleInd].Value - protectorData.Samples[sampleInd-1].Value
			deltas[sampleInd-1] = delta
		}
	}
	for deltaInd := range deltas {
		if deltaInd > 0 {
			if deltas[deltaInd] < deltas[deltaInd-1]*protector.ratio {
				protector.logger.Infof(
					"Matched state degraded. Old value: %.2f, current value: %.2f",
					deltas[deltaInd], deltas[deltaInd-1])
				degraded = true
				break
			}
		}
	}
	if degraded {
		currentState, err := protector.database.GetNotifierState()
		if err != nil {
			protector.logger.Warningf("Can not get notifier state: %s", err.Error())
		}
		if currentState == "OK" {
			if !protector.inspectOnly {
				protector.database.SetNotifierState("ERROR")
			}
		}
	}
	return nil
}
