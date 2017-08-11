package checker

import (
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

	maintenance int64
	isSimple    bool
	ttl         *int64
	ttlState    string
}

func (triggerChecker *TriggerChecker) InitTriggerChecker() (bool, error) {
	triggerChecker.Until = time.Now().Unix()
	trigger, err := triggerChecker.Database.GetTrigger(triggerChecker.TriggerId)
	if err != nil {
		return false, err
	}
	if trigger == nil {
		return false, nil
	}

	triggerChecker.trigger = trigger
	triggerChecker.isSimple = trigger.IsSimpleTrigger

	tagDatas, err := triggerChecker.Database.GetTags(trigger.Tags)
	if err != nil {
		return false, err
	}

	for _, tagData := range tagDatas {
		if tagData.Maintenance != nil && *tagData.Maintenance > triggerChecker.maintenance {
			triggerChecker.maintenance = *tagData.Maintenance
			break
		}
	}

	triggerChecker.ttl = trigger.Ttl
	if trigger.TtlState != nil {
		triggerChecker.ttlState = *trigger.TtlState
	} else {
		triggerChecker.ttlState = NODATA
	}

	triggerChecker.lastCheck, err = getLastCheck(triggerChecker.Database, triggerChecker.TriggerId, triggerChecker.Until)
	if err != nil {
		return false, err
	}

	triggerChecker.From = *triggerChecker.lastCheck.Timestamp
	if triggerChecker.ttl != nil {
		triggerChecker.From = triggerChecker.From - *triggerChecker.ttl
	} else {
		triggerChecker.From = triggerChecker.From - 600
	}

	return true, nil
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
			Timestamp: &emptyLastCheckTimestamp,
		}
	}

	if lastCheck.Timestamp == nil {
		lastCheck.Timestamp = &emptyLastCheckTimestamp
	}

	return lastCheck, nil
}
