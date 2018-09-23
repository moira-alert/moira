package protectors

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/protectors/discover"
	"github.com/moira-alert/moira/protectors/matched"
	"github.com/moira-alert/moira/protectors/random"
)

const (
	discoverMechanism = "discover"
	matchedMechanism  = "matched"
	randomMechanism   = "random"
)

// ConfigureProtector returns protector instance based on given configuration
func ConfigureProtector(protectorConfig map[string]string, database moira.Database,
	logger moira.Logger) (moira.Protector, error) {
	var protector moira.Protector
	var err error
	mechanism := protectorConfig["mechanism"]
	switch mechanism {
	case discoverMechanism, "":
		protector, err = discover.NewProtector(protectorConfig, database, logger)
	case matchedMechanism:
		protector, err = matched.NewProtector(protectorConfig, database, logger)
	case randomMechanism:
		protector, err = random.NewProtector(protectorConfig, database, logger)
	default:
		return nil, fmt.Errorf("unknown mechanism type: %s", mechanism)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid %s protector config: %s", mechanism, err.Error())
	}
	return protector, nil
}
