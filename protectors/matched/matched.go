package matched

import (
	"fmt"
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
	if numSamples < 3 {
		return nil, fmt.Errorf("it takes to collect at least 3 samples to use matched protector")
	}
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
		protectorSamples := make([]float64, 0)
		protectTicker := time.NewTicker(protector.retention)
		for range protectTicker.C {
			if len(protectorSamples) == protector.numSamples {
				protectorData := &ProtectorData{
					values: protectorSamples,
				}
				ch <- protectorData
				protectorSamples = nil
			}
			matched, _ := protector.database.GetMatchedMetricsUpdatesCount()
			protectorSamples = append(protectorSamples, float64(matched))
		}
	}()
	return ch
}

// Protect performs Nodata protection
func (protector *Protector) Protect(protectorData moira.ProtectorData) error {
	var degraded bool
	protectorSamples := protectorData.GetFloats()
	deltas := make([]float64, len(protectorSamples)-1)
	for sampleInd := range protectorSamples {
		if sampleInd > 0 {
			protector.logger.Infof(
				"matched protector value: [old value: %.2f, current value: %.2f]",
				protectorSamples[sampleInd-1],
				protectorSamples[sampleInd],
			)
			delta := protectorSamples[sampleInd] - protectorSamples[sampleInd-1]
			deltas[sampleInd-1] = delta
		}
	}
	for deltaInd := range deltas {
		if deltaInd > 0 {
			protector.logger.Infof(
				"matched protector values delta: [old value: %.2f, current value: %.2f, min_allowed: %.2f]",
				deltas[deltaInd-1],
				deltas[deltaInd],
				deltas[deltaInd-1] * protector.ratio,
			)
			if deltas[deltaInd] < (deltas[deltaInd-1] * protector.ratio) {
				protector.logger.Infof(
					"Matched state degraded. Old value: %.2f, current value: %.2f",
					deltas[deltaInd-1], deltas[deltaInd])
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
