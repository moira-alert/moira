package checker

import (
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/database"
	metricSource "github.com/moira-alert/moira/internal/metric_source"
	"github.com/moira-alert/moira/internal/metrics"
)

// TriggerChecker represents data, used for handling new trigger state
type TriggerChecker struct {
	database moira2.Database
	logger   moira2.Logger
	config   *Config
	metrics  *metrics.CheckMetrics
	source   metricSource.MetricSource

	from  int64
	until int64

	triggerID string
	trigger   *moira2.Trigger
	lastCheck *moira2.CheckData

	ttl      int64
	ttlState moira2.TTLState
}

// MakeTriggerChecker initialize new triggerChecker data
// if trigger does not exists then return ErrTriggerNotExists error
// if trigger metrics source does not configured then return ErrMetricSourceIsNotConfigured error.
func MakeTriggerChecker(triggerID string, dataBase moira2.Database, logger moira2.Logger, config *Config, sourceProvider *metricSource.SourceProvider, metrics *metrics.CheckerMetrics) (*TriggerChecker, error) {
	until := time.Now().Unix()
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

	lastCheck, err := getLastCheck(dataBase, triggerID, until-3600)
	if err != nil {
		return nil, err
	}

	triggerChecker := &TriggerChecker{
		database: dataBase,
		logger:   logger,
		config:   config,
		metrics:  metrics.GetCheckMetrics(&trigger),
		source:   source,

		from:  calculateFrom(lastCheck.Timestamp, trigger.TTL),
		until: until,

		triggerID: triggerID,
		trigger:   &trigger,
		lastCheck: lastCheck,

		ttl:      trigger.TTL,
		ttlState: getTTLState(trigger.TTLState),
	}
	return triggerChecker, nil
}

func getLastCheck(dataBase moira2.Database, triggerID string, emptyLastCheckTimestamp int64) (*moira2.CheckData, error) {
	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil && err != database.ErrNil {
		return nil, err
	}

	if err == database.ErrNil {
		lastCheck = moira2.CheckData{
			Metrics:   make(map[string]moira2.MetricState),
			State:     moira2.StateOK,
			Timestamp: emptyLastCheckTimestamp,
		}
	}

	if lastCheck.Timestamp == 0 {
		lastCheck.Timestamp = emptyLastCheckTimestamp
	}

	return &lastCheck, nil
}

func getTTLState(triggerTTLState *moira2.TTLState) moira2.TTLState {
	if triggerTTLState != nil {
		return *triggerTTLState
	}
	return moira2.TTLStateNODATA
}

func calculateFrom(lastCheckTimestamp, triggerTTL int64) int64 {
	if triggerTTL != 0 {
		return lastCheckTimestamp - triggerTTL
	}
	return lastCheckTimestamp - 600
}
