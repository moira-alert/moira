package protectors

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/protectors/matched"
	"github.com/moira-alert/moira/notifier/protectors/random"
)

const (
	matchedMechanism = "matched"
	randomMechanism  = "random"
)

// ConfigureProtector returns protector instance based on given configuration
func ConfigureProtector(protectorConfig map[string]string,
	database moira.Database, logger moira.Logger) (moira.Protector, []float64) {
	var protector moira.Protector
	var protectorValues []float64
	if mechanism, ok := protectorConfig["mechanism"]; ok {
		switch mechanism {
		case matchedMechanism:
			protector = &matched.Protector{}
		case randomMechanism:
			protector = &random.Protector{}
		default:
			logger.Errorf("Unknown mechanism type: %s", mechanism)
			return nil, nil
		}
		err := protector.Init(protectorConfig, database, logger)
		if err != nil {
			logger.Errorf("Can't configure %s protector: %s", mechanism, err.Error())
			return nil, nil
		}
		protectorValues, err = protector.GetInitialValues()
		if err != nil {
			logger.Errorf("Can't get initial protector values %s protector: %s", mechanism, err.Error())
			return nil, nil
		}
		return protector, protectorValues
	}
	return nil, nil
}
