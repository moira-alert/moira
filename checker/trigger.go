package checker

import (
	"errors"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"time"
)

// TriggerChecker represents data, used for handling new trigger state
type TriggerChecker struct {
	TriggerID string
	Database  moira.Database
	Logger    moira.Logger
	Config    *Config
	Metrics   *graphite.CheckerMetrics

	From  int64
	Until int64

	trigger   *moira.Trigger
	lastCheck *moira.CheckData

	ttl      int64
	ttlState string
}

// ErrTriggerNotExists used if trigger to check does not exists
var ErrTriggerNotExists = errors.New("trigger does not exists")

// InitTriggerChecker initialize new triggerChecker data, if trigger does not exists then return ErrTriggerNotExists error
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
	if triggerChecker.ttl != 0 {
		triggerChecker.From = triggerChecker.From - triggerChecker.ttl
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
