package heartbeat

import (
	"time"

	"github.com/moira-alert/moira"
)

type graphiteLocalChecker struct {
	heartbeat
	count int64
}

func GetGraphiteLocalChecker(delay int64, logger moira.Logger, database moira.Database) Heartbeater {
	if delay > 0 {
		return &graphiteLocalChecker{heartbeat: heartbeat{
			logger:              logger,
			database:            database,
			delay:               delay,
			lastSuccessfulCheck: time.Now().Unix(),
		}}
	}
	return nil
}

func (check *graphiteLocalChecker) Check(nowTS int64) (int64, bool, error) {
	checksCount, err := check.database.GetChecksUpdatesCount()
	if err != nil {
		return 0, false, err
	}

	if check.count != checksCount {
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

func (graphiteLocalChecker) NeedToCheckOthers() bool {
	return true
}

func (check graphiteLocalChecker) NeedTurnOffNotifier() bool {
	checksCont, _ := check.database.GetLocalTriggersToCheckCount()
	return checksCont > 0
}

func (graphiteLocalChecker) GetErrorMessage() string {
	return "Moira-Checker does not check triggers"
}
