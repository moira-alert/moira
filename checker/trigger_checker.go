package checker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metrics"
)

// TriggerChecker represents data, used for handling new trigger state
type TriggerChecker struct {
	database moira.Database
	logger   moira.Logger
	config   *Config
	metrics  *metrics.CheckMetrics
	source   metricSource.MetricSource

	from  int64
	until int64

	triggerID string
	trigger   *moira.Trigger
	lastCheck *moira.CheckData

	ttl      int64
	ttlState moira.TTLState
}

// MakeTriggerChecker initialize new triggerChecker data
// if trigger does not exists then return ErrTriggerNotExists error
// if trigger metrics source does not configured then return ErrMetricSourceIsNotConfigured error.
func MakeTriggerChecker(
	triggerID string,
	dataBase moira.Database,
	logger moira.Logger,
	config *Config,
	sourceProvider *metricSource.SourceProvider,
	metrics *metrics.CheckerMetrics,
) (*TriggerChecker, error) {
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

	lastCheck, err := getLastCheck(dataBase, triggerID, until-3600) //nolint
	if err != nil {
		return nil, err
	}

	triggerLogger := logger.Clone().String(moira.LogFieldNameTriggerID, triggerID)
	if logLevel, ok := config.LogTriggersToLevel[triggerID]; ok {
		if _, err := triggerLogger.Level(logLevel); err != nil {
			triggerLogger.Warning().
				String("log_level", logLevel).
				Msg("Incorrect log level")
		}
	}

	triggerMetrics, _ := metrics.GetCheckMetrics(&trigger)
	triggerChecker := &TriggerChecker{
		database: dataBase,
		logger:   triggerLogger,
		config:   config,
		metrics:  triggerMetrics,
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

func getLastCheck(dataBase moira.Database, triggerID string, emptyLastCheckTimestamp int64) (*moira.CheckData, error) {
	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil && err != database.ErrNil {
		return nil, err
	}

	if err == database.ErrNil {
		lastCheck = moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			State:     moira.StateOK,
			Timestamp: emptyLastCheckTimestamp,
		}
	}

	if lastCheck.Timestamp == 0 {
		lastCheck.Timestamp = emptyLastCheckTimestamp
	}

	return &lastCheck, nil
}

func getTTLState(triggerTTLState *moira.TTLState) moira.TTLState {
	if triggerTTLState != nil {
		return *triggerTTLState
	}
	return moira.TTLStateNODATA
}

func calculateFrom(lastCheckTimestamp, triggerTTL int64) int64 {
	if triggerTTL != 0 {
		return lastCheckTimestamp - triggerTTL
	}
	return lastCheckTimestamp - 600
}
