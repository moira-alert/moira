package heartbeat

import (
	"time"

	"github.com/moira-alert/moira"
)

type filter struct {
	heartbeat
	count                   int64
	firstCheckWasSuccessful bool
}

func GetFilter(delay int64, logger moira.Logger, database moira.Database) Heartbeater {
	if delay > 0 {
		return &filter{heartbeat: heartbeat{
			logger:              logger,
			database:            database,
			delay:               delay,
			lastSuccessfulCheck: time.Now().Unix(),
		},
			firstCheckWasSuccessful: false,
		}
	}
	return nil
}

func (check *filter) Check(nowTS int64) (int64, bool, error) {
	triggersCount, err := check.database.GetLocalTriggersToCheckCount()
	if err != nil {
		return 0, false, err
	}

	metricsCount, err := check.database.GetMetricsUpdatesCount()
	if err != nil {
		return 0, false, err
	}
	if check.count != metricsCount || triggersCount == 0 {
		check.count = metricsCount
		check.lastSuccessfulCheck = nowTS
		return 0, false, nil
	}

	if check.lastSuccessfulCheck < nowTS-check.heartbeat.delay {
		check.logger.Errorb().
			String("error", check.GetErrorMessage()).
			Int64("time_since_successful_check", nowTS-check.heartbeat.lastSuccessfulCheck).
			Msg("Send message")

		check.firstCheckWasSuccessful = true
		return nowTS - check.heartbeat.lastSuccessfulCheck, true, nil
	}
	return 0, false, nil
}

// NeedTurnOffNotifier: turn off notifications if at least once the filter check was successful
func (check filter) NeedTurnOffNotifier() bool {
	return check.firstCheckWasSuccessful
}

func (check filter) NeedToCheckOthers() bool {
	return true
}

func (filter) GetErrorMessage() string {
	return "Moira-Filter does not receive metrics"
}
