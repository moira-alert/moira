package selfstate

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/protectors/undefined"
	"github.com/moira-alert/moira/protectors/matched"
	"github.com/moira-alert/moira/protectors/random"
)

// Notifier states
const (
	OK    = "OK"    // OK means notifier is healthy
	ERROR = "ERROR" // ERROR means notifier is stopped, admin intervention is required
)

// Nodata protector mechanisms
const (
	matchedMechanism = "matched"
	randomMechanism  = "random"
)

// ConfigureProtector returns protector instance based on given configuration
func ConfigureProtector(config Config, database moira.Database, logger moira.Logger) (moira.Protector, error) {
	var protector moira.Protector
	var err error
	switch config.ProtectorMechanism {
	case matchedMechanism:
		protector, err = matched.NewProtector(database, logger, config.ProtectorInspectOnly, config.ProtectorNumSamples,
			config.ProtectorSampleRetention, config.ProtectorSampleRatio, config.ProtectorThrottling)
	case randomMechanism:
		protector, err = random.NewProtector(database, logger, config.ProtectorInspectOnly, config.ProtectorNumSamples,
			config.ProtectorSampleRetention, config.ProtectorSampleRatio, config.ProtectorThrottling)
	default:
		protector = undefined.GetDefaultProtector()
		return protector, nil
	}
	if err != nil {
		return nil, fmt.Errorf("invalid %s protector config: %s", config.ProtectorMechanism, err.Error())
	}
	return protector, nil
}
