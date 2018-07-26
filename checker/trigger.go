package checker

import (
	"errors"
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics/graphite"
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

	if trigger.TriggerType == "" {
		if err := updateEmptyTriggerType(&trigger, triggerChecker.Database, triggerChecker.Logger); err != nil {
			return err
		}
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

func updateEmptyTriggerType(trigger *moira.Trigger, dataBase moira.Database, logger moira.Logger) error {
	if err := setProperTriggerType(trigger, logger); err == nil {
		logger.Infof("Trigger %v - save to Database", trigger.ID)
		if err := dataBase.SaveTrigger(trigger.ID, trigger); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("trigger converter: trigger %v - could not save to Database, error: %v",
			trigger.ID, err)
	}
	return nil
}

func setProperTriggerType(trigger *moira.Trigger, logger moira.Logger) error {
	logger.Infof("Trigger %v has empty trigger_type, start conversion", trigger.ID)
	if trigger.Expression != nil && *trigger.Expression != "" {
		logger.Infof("Trigger %v has expression '%v', switch to %v...",
			trigger.ID, *trigger.Expression, moira.ExpressionTrigger)
		trigger.TriggerType = moira.ExpressionTrigger
	}

	if trigger.WarnValue != nil && trigger.ErrorValue != nil {
		logger.Infof("Trigger %v - warn_value: %v, error_value: %v",
			trigger.ID, trigger.WarnValue, trigger.ErrorValue)
		if *trigger.ErrorValue > *trigger.WarnValue {
			logger.Infof("Trigger %v - set trigger_type to %v", trigger.ID, moira.RisingTrigger)
			trigger.TriggerType = moira.RisingTrigger
			return nil
		}
		if *trigger.ErrorValue < *trigger.WarnValue {
			logger.Infof("Trigger %v - set trigger_type to %v", trigger.ID, moira.FallingTrigger)
			trigger.TriggerType = moira.FallingTrigger
			return nil
		}
		if *trigger.ErrorValue == *trigger.WarnValue {
			logger.Infof("Trigger %v - warn_value == error_value, set trigger_type to %v, set warn_value to null",
				trigger.ID, moira.RisingTrigger)
			trigger.TriggerType = moira.RisingTrigger
			trigger.WarnValue = nil
			return nil
		}
	}

	return fmt.Errorf("cannot update trigger %v - warn_value: %v, error_value: %v, expression: %v, trigger_type: ''",
		trigger.ID, trigger.WarnValue, trigger.ErrorValue, trigger.Expression)
}
