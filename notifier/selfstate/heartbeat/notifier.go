package heartbeat

import (
	"fmt"

	"github.com/moira-alert/moira"
)

type notifier struct {
	heartbeat
}

func GetNotifier(checkTags []string, logger moira.Logger, database moira.Database) Heartbeater {
	return &notifier{heartbeat{
		database:  database,
		logger:    logger,
		checkTags: checkTags,
	}}
}

func (check notifier) Check(int64) (int64, bool, error) {
	state, _ := check.database.GetNotifierState()
	if state.State != moira.SelfStateOK && state.Actor == moira.SelfStateActorManual {
		check.logger.Error().
			String("error", check.GetErrorMessage()).
			Msg("Notifier is not healthy")

		return 0, true, nil
	}

	check.logger.Debug().
		String("state", state.State).
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
	state, _ := check.database.GetNotifierState()
	return fmt.Sprintf("Moira-Notifier does not send messages. State: %v", state.State)
}

func (check *notifier) GetCheckTags() CheckTags {
	return check.checkTags
}
