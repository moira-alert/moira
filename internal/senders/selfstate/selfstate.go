package selfstate

import (
	"fmt"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

// Sender implements moira sender interface via selfstate
type Sender struct {
	Database moira2.Database
	logger   moira2.Logger
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira2.Logger, location *time.Location, dateTimeFormat string) error {
	sender.logger = logger
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) error {
	selfState, err := sender.Database.GetNotifierState()
	if err != nil {
		return fmt.Errorf("failed to get notifier state: %s", err.Error())
	}
	subjectState := events.GetSubjectState()
	switch subjectState {
	case moira2.StateTEST:
		sender.logger.Infof("current notifier state: %s", selfState)
		return nil
	case moira2.StateOK, moira2.StateEXCEPTION:
		sender.logger.Errorf("[trigger: %s] state %s is ignorable", trigger.ID, subjectState.String())
		return nil
	default:
		if selfState != subjectState.ToSelfState() {
			if err := sender.Database.SetNotifierState(moira2.SelfStateERROR); err != nil {
				return fmt.Errorf("failed to disable notifications: %s", err.Error())
			}
		}
	}
	return nil
}
