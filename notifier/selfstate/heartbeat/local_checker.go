package heartbeat

import (
	"time"

	"github.com/moira-alert/moira"
)

type localChecker struct {
	heartbeat
	count int64
}

func GetLocalChecker(delay int64, logger moira.Logger, database moira.Database) Heartbeater {
	if delay > 0 {
		return &localChecker{heartbeat: heartbeat{
			logger:              logger,
			database:            database,
			delay:               delay,
			lastSuccessfulCheck: time.Now().Unix(),
		}}
	}
	return nil
}

func (check *localChecker) Check(nowTS int64) (int64, bool, error) {
	triggersCount, err := check.database.GetLocalTriggersToCheckCount()
	if err != nil {
		return 0, false, err
	}

	checksCount, _ := check.database.GetChecksUpdatesCount()
	if check.count != checksCount || triggersCount == 0 {
		check.count = checksCount
		check.lastSuccessfulCheck = nowTS
		return 0, false, nil
	}

	if check.lastSuccessfulCheck < nowTS-check.delay {
		check.logger.Errorf(templateMoreThanMessage, check.GetErrorMessage(), nowTS-check.lastSuccessfulCheck)
		return nowTS - check.lastSuccessfulCheck, true, nil
	}

	return 0, false, nil
}

func (localChecker) NeedToCheckOthers() bool {
	return true
}

func (check localChecker) NeedTurnOffNotifier() bool {
	return false
}

func (localChecker) GetErrorMessage() string {
	return "Moira-Checker does not check triggers"
}
