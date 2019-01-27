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
	database moira.Database
	logger   moira.Logger
	config   *Config
	metrics  *graphite.CheckMetrics
	source   metricSource.MetricSource

	from  int64
	until int64

	triggerID string
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
		triggerID: triggerID,
		database:  dataBase,
		logger:    logger,
		config:    config,
		metrics:   metrics.GetCheckMetrics(&trigger),
		source:    source,
		until:     time.Now().Unix(),
		trigger:   &trigger,
		ttl:       trigger.TTL,
	}

	if trigger.TTLState != nil {
		triggerChecker.ttlState = *trigger.TTLState
	} else {
		triggerChecker.ttlState = NODATA
	}

	lastCheck, err := getLastCheck(triggerChecker.database, triggerChecker.triggerID, triggerChecker.until-3600)
	if err != nil {
		return nil, err
	}
	triggerChecker.lastCheck = lastCheck

	triggerChecker.from = triggerChecker.lastCheck.Timestamp
	if triggerChecker.ttl != 0 {
		triggerChecker.from = triggerChecker.from - triggerChecker.ttl
	} else {
		triggerChecker.from = triggerChecker.from - 600
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
