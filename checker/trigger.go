package checker

import (
	"errors"
	"github.com/moira-alert/moira-alert"
	"time"
)

type TriggerChecker struct {
	TriggerId string
	Database  moira.Database
	Logger    moira.Logger
	Config    *Config

	From  int64
	Until int64

	trigger   *moira.Trigger
	lastCheck *moira.CheckData

	isSimple    bool
	ttl         *int64
	ttlState    string
}

var ErrTriggerNotExists = errors.New("trigger does not exists")

func (triggerChecker *TriggerChecker) InitTriggerChecker() error {
	triggerChecker.Until = time.Now().Unix()
	trigger, err := triggerChecker.Database.GetTrigger(triggerChecker.TriggerId)
	if err != nil {
		return err
	}
	if trigger == nil {
		return ErrTriggerNotExists
	}

	triggerChecker.trigger = trigger
	triggerChecker.isSimple = trigger.IsSimpleTrigger
	triggerChecker.ttl = trigger.Ttl

	if trigger.TtlState != nil {
		triggerChecker.ttlState = *trigger.TtlState
	} else {
		triggerChecker.ttlState = NODATA
	}

	triggerChecker.lastCheck, err = getLastCheck(triggerChecker.Database, triggerChecker.TriggerId, triggerChecker.Until-3600)
	if err != nil {
		return err
	}

	triggerChecker.From = triggerChecker.lastCheck.Timestamp
	if triggerChecker.ttl != nil {
		triggerChecker.From = triggerChecker.From - *triggerChecker.ttl
	} else {
		triggerChecker.From = triggerChecker.From - 600
	}

	return nil
}

func getLastCheck(database moira.Database, triggerId string, emptyLastCheckTimestamp int64) (*moira.CheckData, error) {
	lastCheck, err := database.GetTriggerLastCheck(triggerId)
	if err != nil {
		return lastCheck, err
	}

	if lastCheck == nil {
		lastCheck = &moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			State:     NODATA,
			Timestamp: emptyLastCheckTimestamp,
		}
	}

	if lastCheck.Timestamp == 0 {
		lastCheck.Timestamp = emptyLastCheckTimestamp
	}

	return lastCheck, nil
}
