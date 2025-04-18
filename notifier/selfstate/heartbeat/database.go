package heartbeat

import "github.com/moira-alert/moira"

type databaseHeartbeat struct{ heartbeat }

func GetDatabase(delay, lastSuccessfulCheck int64, checkTags []string, logger moira.Logger, database moira.Database) Heartbeater {
	if delay > 0 {
		return &databaseHeartbeat{heartbeat{
			logger:              logger,
			database:            database,
			delay:               delay,
			lastSuccessfulCheck: lastSuccessfulCheck,
			checkTags:           checkTags,
		}}
	}

	return nil
}

func (check *databaseHeartbeat) Check(nowTS int64) (int64, bool, error) {
	_, err := check.database.GetChecksUpdatesCount()
	if err == nil {
		check.lastSuccessfulCheck = nowTS
		return 0, false, nil
	}

	if check.lastSuccessfulCheck < nowTS-check.delay {
		check.logger.Error().
			String("error", check.GetErrorMessage()).
			Int64("time_since_successful_check", nowTS-check.heartbeat.lastSuccessfulCheck).
			Msg("Send message")

		return nowTS - check.lastSuccessfulCheck, true, nil
	}

	return 0, false, nil
}

func (databaseHeartbeat) NeedTurnOffNotifier() bool {
	return true
}

func (databaseHeartbeat) NeedToCheckOthers() bool {
	return false
}

func (databaseHeartbeat) GetErrorMessage() string {
	return "Redis disconnected"
}

func (check *databaseHeartbeat) GetCheckTags() CheckTags {
	return check.checkTags
}
