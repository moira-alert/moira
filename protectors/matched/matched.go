package matched

import (
	"time"

	"github.com/gosexy/to"
	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	database  moira.Database
	logger    moira.Logger
	matchedK  float64
	interval  time.Duration
	points    int
	badPoints int
	dryRun    bool
}

// NewProtector returns new protector
func NewProtector(protectorConfig moira.ProtectorConfig, database moira.Database, logger moira.Logger) (*Protector, error) {
	return &Protector{
		database:  database,
		logger:    logger,
		matchedK:  protectorConfig.Threshold,
		interval:  to.Duration(protectorConfig.FetchInterval),
		points:    protectorConfig.PointsToFetch,
		badPoints: protectorConfig.MaxBadPoints,
		dryRun:    protectorConfig.DryRunMode,
	}, nil
}

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() <-chan moira.ProtectorData {
	ch := make(chan moira.ProtectorData)
	go func() {
		protectorSamples := make([]moira.ProtectorSample, 0)
		protectTicker := time.NewTicker(protector.interval)
		for t := range protectTicker.C {
			if len(protectorSamples) == protector.points {
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
			if deltas[deltaInd] < deltas[deltaInd-1]*protector.matchedK {
				protector.logger.Infof(
					"Matched state degraded. Old value: %.2f, current value: %.2f",
					deltas[deltaInd], deltas[deltaInd-1])
				degraded = true
				break
			}
		}
	}
	if degraded && protector.dryRun {
		currentState, err := protector.database.GetNotifierState()
		if err != nil {
			protector.logger.Warningf("Can not get notifier state: %s", err.Error())
		}
		if currentState == "OK" {
			protector.database.SetNotifierState("ERROR")
		}
	}
	return nil
}
