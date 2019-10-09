package selfstate

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface via selfstate
type Sender struct {
	Database moira.Database
	logger   moira.Logger
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.logger = logger
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
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
		sender.logger.Errorf("[trigger: %s] state %s is ignorable", trigger.ID, subjectState.String())
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
