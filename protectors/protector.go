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

// ConfigureProtector returns protector instance based on given configuration
func ConfigureProtector(protectorConfig map[string]string,
	database moira.Database, logger moira.Logger) (moira.Protector, error) {
	var protector moira.Protector
	if mechanism, ok := protectorConfig["mechanism"]; ok {
		switch mechanism {
		case matchedMechanism:
			protector = &matched.Protector{}
		case randomMechanism:
			protector = &random.Protector{}
		default:
			return nil, fmt.Errorf("unknown mechanism type: %s", mechanism)
		}
		err := protector.Init(protectorConfig, database, logger)
		if err != nil {
			return nil, fmt.Errorf("can't configure %s protector: %s", mechanism, err.Error())
		}
	}
	return protector, nil
}
