package heartbeat

import "github.com/moira-alert/moira"

type remoteChecker struct {
	heartbeat
	count int64
}

func GetRemoteChecker(delay, lastSuccessfulCheck int64, checkTags []string, logger moira.Logger, database moira.Database) Heartbeater {
	if delay > 0 {
		return &remoteChecker{heartbeat: heartbeat{
			logger:              logger,
			database:            database,
			delay:               delay,
			lastSuccessfulCheck: lastSuccessfulCheck,
			checkTags:           checkTags,
		}}
	}

	return nil
}

func (check *remoteChecker) Check(nowTS int64) (int64, bool, error) {
	defaultRemoteCluster := moira.DefaultGraphiteRemoteCluster

	triggerCount, err := check.database.GetTriggersToCheckCount(defaultRemoteCluster)
	if err != nil {
		return 0, false, err
	}

	remoteTriggersCount, _ := check.database.GetRemoteChecksUpdatesCount()
	if check.count != remoteTriggersCount || triggerCount == 0 {
		check.count = remoteTriggersCount
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

func (check remoteChecker) NeedTurnOffNotifier() bool {
	return false
}

func (remoteChecker) NeedToCheckOthers() bool {
	return true
}

func (remoteChecker) GetErrorMessage() string {
	return "Moira-Remote-Checker does not check remote triggers"
}

func (check *remoteChecker) GetCheckTags() CheckTags {
	return check.checkTags
}
