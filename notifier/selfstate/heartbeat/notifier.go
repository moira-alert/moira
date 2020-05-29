package heartbeat

import (
	"fmt"

	"github.com/moira-alert/moira"
)

type notifier struct {
	db  moira.Database
	log moira.Logger
}

func GetNotifier(logger moira.Logger, database moira.Database) Heartbeater {
	return &notifier{
		db:  database,
		log: logger,
	}
}

func (check notifier) Check(int64) (int64, bool, error) {
	if state, _ := check.db.GetNotifierState(); state != moira.SelfStateOK {
		check.log.Errorf("%s. Send message.", check.GetErrorMessage())
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
	return fmt.Sprintf("Moira-Notifier does not send messages. State: %v", state)
}
