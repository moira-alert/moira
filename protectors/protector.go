package protectors

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/protectors/matched"
	"github.com/moira-alert/moira/protectors/random"
)

const (
	matchedMechanism = "matched"
	randomMechanism  = "random"
)

// GetDefaultProtector returns default Nodata protector
func GetDefaultProtector() *Protector {
	return &Protector{}
}

// Protector implements NoData Protector interface
type Protector struct {}

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() <-chan moira.ProtectorData {
	ch := make(chan moira.ProtectorData)
	go func() {
		for {
			protectorData := moira.ProtectorData{}
			ch <- protectorData
		}
	}()
	return ch
}

// Protect performs Nodata protection
func (protector *Protector) Protect(protectorData moira.ProtectorData) error {
	return nil
}

// ConfigureProtector returns protector instance based on given configuration
func ConfigureProtector(protectorConfig moira.ProtectorConfig, database moira.Database,
	logger moira.Logger) (moira.Protector, error) {
	var protector moira.Protector
	var err error
	switch protectorConfig.Mechanism {
	case matchedMechanism:
		protector, err = matched.NewProtector(protectorConfig, database, logger)
	case randomMechanism:
		protector, err = random.NewProtector(protectorConfig, database, logger)
	default:
		protector = GetDefaultProtector()
		return protector, nil
	}
	if err != nil {
		return nil, fmt.Errorf("invalid %s protector config: %s",
			protectorConfig.Mechanism, err.Error())
	}
	return protector, nil
}
