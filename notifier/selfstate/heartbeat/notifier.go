package heartbeat

import (
	"fmt"

	"github.com/moira-alert/moira/metrics"

	"github.com/moira-alert/moira"
)

type notifier struct {
	db              moira.Database
	log             moira.Logger
	notifierIsAlive metrics.Meter
}

func GetNotifier(logger moira.Logger, database moira.Database, notifierIsAlive metrics.Meter) Heartbeater {
	return &notifier{
		db:              database,
		log:             logger,
		notifierIsAlive: notifierIsAlive,
	}
}

func (check notifier) Check(int64) (int64, bool, error) {
	if state, _ := check.db.GetNotifierState(); state != moira.SelfStateOK {
		check.log.Error().
			String("error", check.GetErrorMessage()).
			Msg("Notifier is not healthy")

		return 0, true, nil
	}

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
	alive := int64(0)
	if state == moira.SelfStateOK {
		alive = 1
	}

	check.notifierIsAlive.Mark(alive)

	return fmt.Sprintf("Moira-Notifier does not send messages. State: %v", state)
}
