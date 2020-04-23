package heartbeat

import (
	"time"

	"github.com/moira-alert/moira"
)

type graphiteRemoteChecker struct {
	heartbeat
	count int64
}

func GetGraphiteRemoteChecker(delay int64, logger moira.Logger, database moira.Database) Heartbeater {
	if delay > 0 {
		return &graphiteRemoteChecker{heartbeat: heartbeat{
			logger:              logger,
			database:            database,
			delay:               delay,
			lastSuccessfulCheck: time.Now().Unix(),
		}}
	}
	return nil
}

func (check *graphiteRemoteChecker) Check(nowTS int64) (int64, bool, error) {
	remoteTriggersCount, err := check.database.GetRemoteChecksUpdatesCount()
	if err != nil {
		return 0, false, err
	}

	if check.count != remoteTriggersCount {
		check.count = remoteTriggersCount
		check.lastSuccessfulCheck = nowTS
		return 0, false, nil
	}

	if check.lastSuccessfulCheck < nowTS-check.delay {
		check.logger.Errorf(templateMoreThanMessage, check.GetErrorMessage())
		return nowTS - check.lastSuccessfulCheck, true, nil
	}
	return 0, false, nil
}

func (check graphiteRemoteChecker) NeedTurnOffNotifier() bool {
	remoteTriggersCount, _ := check.database.GetRemoteTriggersToCheckCount()
	return remoteTriggersCount > 0
}

func (graphiteRemoteChecker) NeedToCheckOthers() bool {
	return true
}

func (graphiteRemoteChecker) GetErrorMessage() string {
	return "Moira-Remote-Checker does not check remote triggers"
}
