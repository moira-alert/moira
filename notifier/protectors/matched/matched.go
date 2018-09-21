package matched

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
	matchedK float64
}

// Init configures protector
func (protector *Protector) Init(protectorSettings map[string]string, database moira.Database, logger moira.Logger) error {
	var err error
	protector.database = database
	protector.logger = logger
	protector.matchedK, err = strconv.ParseFloat(protectorSettings["k"], 64)
	if err != nil {
		return fmt.Errorf("can not read matched k from config: %s", err.Error())
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
			matched, _ := protector.database.GetMatchedMetricsUpdatesCount()
			protectorData = append(protectorData, moira.ProtectorData{
				Value:float64(matched),
				})
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
	degraded := protectorData[1].Value < protectorData[0].Value * float64(protector.matchedK)
	if degraded {
		protector.logger.Infof(
			"Matched state degraded. Old value: %.2f, current value: %.2f",
			protectorData[0].Value, protectorData[1].Value)
	}
	return degraded
}
