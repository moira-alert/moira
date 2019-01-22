package checker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metrics/graphite"
)

// TriggerChecker represents data, used for handling new trigger state
type TriggerChecker struct {
	TriggerID string
	Database  moira.Database
	Logger    moira.Logger
	Config    *Config
	Metrics   *graphite.CheckMetrics
	Source    metricSource.MetricSource

	From  int64
	Until int64

	trigger   *moira.Trigger
	lastCheck *moira.CheckData

	ttl      int64
	ttlState string
}

// MakeTriggerChecker initialize new triggerChecker data, if trigger does not exists then return ErrTriggerNotExists error
func MakeTriggerChecker(triggerID string, dataBase moira.Database, logger moira.Logger, config *Config, sourceProvider *metricSource.SourceProvider, metrics *graphite.CheckerMetrics) (*TriggerChecker, error) {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil, ErrTriggerNotExists
		}
		return nil, err
	}

	source, err := sourceProvider.GetTriggerMetricSource(&trigger)
	if err != nil {
		return nil, err
	}

	triggerChecker := TriggerChecker{
		TriggerID: triggerID,
		Database:  dataBase,
		Logger:    logger,
		Config:    config,
		Metrics:   metrics.GetCheckMetrics(&trigger),
		Source:    source,
		Until:     time.Now().Unix(),
		trigger:   &trigger,
		ttl:       trigger.TTL,
	}

	if trigger.TTLState != nil {
		triggerChecker.ttlState = *trigger.TTLState
	} else {
		triggerChecker.ttlState = NODATA
	}

	lastCheck, err := getLastCheck(triggerChecker.Database, triggerChecker.TriggerID, triggerChecker.Until-3600)
	if err != nil {
		return nil, err
	}
	triggerChecker.lastCheck = lastCheck

	triggerChecker.From = triggerChecker.lastCheck.Timestamp
	if triggerChecker.ttl != 0 {
		triggerChecker.From = triggerChecker.From - triggerChecker.ttl
	} else {
		triggerChecker.From = triggerChecker.From - 600
	}

	return &triggerChecker, nil
}

func getLastCheck(dataBase moira.Database, triggerID string, emptyLastCheckTimestamp int64) (*moira.CheckData, error) {
	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil && err != database.ErrNil {
		return nil, err
	}

	if err == database.ErrNil {
		lastCheck = moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			State:     OK,
			Timestamp: emptyLastCheckTimestamp,
		}
	}

	if lastCheck.Timestamp == 0 {
		lastCheck.Timestamp = emptyLastCheckTimestamp
	}

	return &lastCheck, nil
}
