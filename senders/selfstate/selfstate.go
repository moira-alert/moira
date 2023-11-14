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
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string, _ moira.Database) error {
	sender.logger = logger
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	selfState, err := sender.Database.GetNotifierState()
	if err != nil {
		return fmt.Errorf("failed to get notifier state: %s", err.Error())
	}

	state := events.GetCurrentState(throttled)

	switch state {
	case moira.StateTEST:
		sender.logger.Info().
			String("notifier_state", selfState).
			Msg("Current notifier state")

		return nil
	case moira.StateOK, moira.StateEXCEPTION:
		sender.logger.Error().
			String(moira.LogFieldNameTriggerID, trigger.ID).
			String("state", state.String()).
			Msg("State is ignorable")

		return nil
	default:
		if selfState != state.ToSelfState() {
			if err := sender.Database.SetNotifierState(moira.SelfStateERROR); err != nil {
				return fmt.Errorf("failed to disable notifications: %s", err.Error())
			}
		}
	}

	return nil
}
