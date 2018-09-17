package protectors

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/protectors/matched"
)

// ConfigureProtector returns protector instance based on given configuration
func ConfigureProtector(protectorConfig map[string]string, database moira.Database, logger moira.Logger) (moira.Protector, []int64) {
	var protector moira.Protector
	if strategy, ok := protectorConfig["strategy"]; ok {
		switch strategy {
		case "matched":
			protector = &matched.Protector{}
			protector.Init(
				protectorConfig,
				database,
				logger,
			)
			return protector, protector.GetInitialValues()
		}
	}
	return nil, []int64{}
}
