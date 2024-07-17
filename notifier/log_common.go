package notifier

import "github.com/moira-alert/moira"

func getLogWithPackageContext(log *moira.Logger, pkg *NotificationPackage, config *Config) moira.Logger {
	logger := (*log).Clone().
		String(moira.LogFieldNameContactID, pkg.Contact.ID).
		String(moira.LogFieldNameContactType, pkg.Contact.Type).
		String(moira.LogFieldNameContactValue, pkg.Contact.Value).
		Int(moira.LogFieldNameFailCount, pkg.FailCount).
		String(moira.LogFieldNameContactUser, pkg.Contact.User).
		String(moira.LogFieldNameContactTeam, pkg.Contact.Team)
	if pkg.Trigger.ID != "" { // note: test notification without trigger info
		logger.
			String(moira.LogFieldNameTriggerID, pkg.Trigger.ID).
			String(moira.LogFieldNameTriggerName, pkg.Trigger.Name)
	}

	SetLogLevelByConfig(config.LogContactsToLevel, pkg.Contact.ID, &logger)
	return logger
}

func SetLogLevelByConfig(entityToLevel map[string]string, entityID string, logger *moira.Logger) {
	if v, ok := entityToLevel[entityID]; ok {
		if _, err := (*logger).Level(v); err != nil {
			(*logger).Warning().
				Error(err).
				Msg("Couldn't set log level")
		}
	}
}
