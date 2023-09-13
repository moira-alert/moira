package heartbeat

import (
	"fmt"

	"github.com/moira-alert/moira/metrics"

	"github.com/moira-alert/moira"
)

type notifier struct {
	db      moira.Database
	log     moira.Logger
	metrics *metrics.HeartBeatMetrics
}

func GetNotifier(logger moira.Logger, database moira.Database, metrics *metrics.HeartBeatMetrics) Heartbeater {
	return &notifier{
		db:      database,
		log:     logger,
		metrics: metrics,
	}
}

func (check notifier) Check(int64) (int64, bool, error) {
	state, _ := check.db.GetNotifierState()
	if state != moira.SelfStateOK {
		check.metrics.NotifierIsAlive.Mark(0)

		check.log.Error().
			String("error", check.GetErrorMessage()).
			Msg("Notifier is not healthy")

		return 0, true, nil
	}
	check.metrics.NotifierIsAlive.Mark(1)

	check.log.Debug().
		String("state", state).
		Msg("Notifier is healthy")

	return 0, false, nil
}

func (notifier) NeedTurnOffNotifier() bool {
	return false
}

func (notifier) NeedToCheckOthers() bool {
	return true
}

func (check notifier) GetErrorMessage() string {
	state, _ := check.db.GetNotifierState()
	return fmt.Sprintf("Moira-Notifier does not send messages. State: %v", state)
}
