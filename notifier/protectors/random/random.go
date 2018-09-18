package random

import (
	"fmt"
	"strconv"

	"github.com/moira-alert/moira"
)

// Protector implements NoData Protector interface
type Protector struct {
	enabled  bool
	database moira.Database
	logger   moira.Logger
	randomK  float64
}

// Init configures protector
func (protector *Protector) Init(protectorSettings map[string]string, database moira.Database, logger moira.Logger) error {
	var err error
	protector.database = database
	protector.logger = logger
	protector.randomK, err = strconv.ParseFloat(protectorSettings["k"], 64)
	if err != nil {
		return fmt.Errorf("can not read random k from config: %s", err.Error())
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
	return []float64{0, 0}, nil
}

// GetCurrentValues returns current values based on previously taken values
func (protector *Protector) GetCurrentValues(oldValues []float64) ([]float64, error) {
	return nil, nil
}

// IsStateDegraded returns true if state is degraded
//func (protector *Protector) IsStateDegraded(oldValues []float64, currentValues []float64) bool {
	//degraded :=
	//if degraded {
	//	protector.logger.Infof(
	//		"Matched state degraded. Old value: %.2f, current value: %.2f",
	//		oldValues[1], currentValues[1])
	//}
	//return degraded
//}

