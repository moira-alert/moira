package discover

import (
	"time"

	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	database moira.Database
	logger   moira.Logger
}

// NewProtector returns new protector
func NewProtector(protectorSettings map[string]string, database moira.Database, logger moira.Logger) (*Protector, error) {
	return &Protector{
		database: database,
		logger:   logger,
	}, nil
}

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() <-chan moira.ProtectorData {
	ch := make(chan moira.ProtectorData)
	go func() {
		protectorSamples := make([]moira.ProtectorSample, 0)
		for {
			total, _ := protector.database.GetMetricsUpdatesCount()
			protectorSamples = append(protectorSamples, moira.ProtectorSample{
				Name: "total_metrics",
				Value: float64(total),
			})
			if len(protectorSamples) == 10 {
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
	return nil
}
