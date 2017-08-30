package checker

import (
	"errors"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"time"
)

type TriggerChecker struct {
	TriggerID string
	Database  moira.Database
	Logger    moira.Logger
	Config    *Config

	From  int64
	Until int64

	trigger   *moira.Trigger
	lastCheck *moira.CheckData

	isSimple bool
	ttl      *int64
	ttlState string
}

var ErrTriggerNotExists = errors.New("trigger does not exists")

func (triggerChecker *TriggerChecker) InitTriggerChecker() error {
	triggerChecker.Until = time.Now().Unix()
	trigger, err := triggerChecker.Database.GetTrigger(triggerChecker.TriggerID)
	if err != nil {
		if err == database.ErrNil {
			return ErrTriggerNotExists
		}
		return err
	}

	triggerChecker.trigger = &trigger
	triggerChecker.isSimple = trigger.IsSimpleTrigger
	triggerChecker.ttl = trigger.TTL

	if trigger.TTLState != nil {
		triggerChecker.ttlState = *trigger.TTLState
	} else {
		triggerChecker.ttlState = NODATA
	}

	triggerChecker.lastCheck, err = getLastCheck(triggerChecker.Database, triggerChecker.TriggerID, triggerChecker.Until-3600)
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

func getLastCheck(dataBase moira.Database, triggerID string, emptyLastCheckTimestamp int64) (*moira.CheckData, error) {
	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil && err != database.ErrNil {
		return nil, err
	}

	if err == database.ErrNil {
		lastCheck = moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			State:     NODATA,
			Timestamp: emptyLastCheckTimestamp,
		}
	}

	if lastCheck.Timestamp == 0 {
		lastCheck.Timestamp = emptyLastCheckTimestamp
	}

	return &lastCheck, nil
}
