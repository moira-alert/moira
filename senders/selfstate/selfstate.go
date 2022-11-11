package selfstate

import (
	"fmt"

	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface via selfstate.
// Use NewSender to create instance.
type Sender struct {
	Database moira.Database
	logger   moira.Logger
}

// NewSender creates Sender instance.
func NewSender(logger moira.Logger, db moira.Database) *Sender {
	sender := &Sender{
		Database: db,
	}

	sender.logger = logger

	return sender
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, _ moira.ContactData, trigger moira.TriggerData, _ [][]byte, _ bool) error {
	selfState, err := sender.Database.GetNotifierState()
	if err != nil {
		return fmt.Errorf("failed to get notifier state: %s", err.Error())
	}
	subjectState := events.GetSubjectState()
	switch subjectState {
	case moira.StateTEST:
		sender.logger.Infof("current notifier state: %s", selfState)
		return nil
	case moira.StateOK, moira.StateEXCEPTION:
		sender.logger.Clone().String(moira.LogFieldNameTriggerID, trigger.ID).
			Errorf("state %s is ignorable", subjectState.String())
		return nil
	default:
		if selfState != subjectState.ToSelfState() {
			if err := sender.Database.SetNotifierState(moira.SelfStateERROR); err != nil {
				return fmt.Errorf("failed to disable notifications: %s", err.Error())
			}
		}
	}
	return nil
}
