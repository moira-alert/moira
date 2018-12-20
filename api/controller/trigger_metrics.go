package controller

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/remote"
	"github.com/moira-alert/moira/target"
)

// GetTriggerEvaluationResult evaluates every target in trigger and returns
// result, separated on main and additional targets metrics
func GetTriggerEvaluationResult(dataBase moira.Database, remoteConfig *remote.Config,
	from, to int64, triggerID string) (*checker.TriggerTimeSeries, *moira.Trigger, error) {
	allowRealtimeAlerting := true
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		return nil, nil, err
	}
	triggerMetrics := &checker.TriggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}
	if trigger.IsRemote && !remoteConfig.IsEnabled() {
		return nil, &trigger, remote.ErrRemoteStorageDisabled
	}
	for i, tar := range trigger.Targets {
		var timeSeries []*target.TimeSeries
		if trigger.IsRemote {
			timeSeries, err = remote.Fetch(remoteConfig, tar, from, to, allowRealtimeAlerting)
			if err != nil {
				return nil, &trigger, err
			}
		} else {
			result, err := target.EvaluateTarget(dataBase, tar, from, to, allowRealtimeAlerting)
			if err != nil {
				return nil, &trigger, err
			}
			timeSeries = result.TimeSeries
		}
		if i == 0 {
			triggerMetrics.Main = timeSeries
		} else {
			triggerMetrics.Additional = append(triggerMetrics.Additional, timeSeries...)
		}
	}
	return triggerMetrics, &trigger, nil
}

// DeleteTriggerMetric deletes metric from last check and all trigger patterns metrics
func DeleteTriggerMetric(dataBase moira.Database, metricName string, triggerID string) *api.ErrorResponse {
	return deleteTriggerMetrics(dataBase, metricName, triggerID, false)
}

// DeleteTriggerNodataMetrics deletes all metric from last check which are in NODATA state.
// It also deletes all trigger patterns of those metrics
func DeleteTriggerNodataMetrics(dataBase moira.Database, triggerID string) *api.ErrorResponse {
	return deleteTriggerMetrics(dataBase, "", triggerID, true)
}

// GetTriggerMetrics gets all trigger metrics values, default values from: now - 10min, to: now
func GetTriggerMetrics(dataBase moira.Database, remoteConfig *remote.Config,
	from, to int64, triggerID string) (*dto.TriggerMetrics, *api.ErrorResponse) {
	tts, _, err := GetTriggerEvaluationResult(dataBase, remoteConfig, from, to, triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger not found"))
		}
		return nil, api.ErrorInternalServer(err)
	}
	triggerMetrics := dto.TriggerMetrics{
		Main:       make(map[string][]*moira.MetricValue),
		Additional: make(map[string][]*moira.MetricValue),
	}
	for _, timeSeries := range tts.Main {
		values := make([]*moira.MetricValue, 0)
		for i := 0; i < len(timeSeries.Values); i++ {
			timestamp := timeSeries.StartTime + int64(i)*timeSeries.StepTime
			value := timeSeries.GetTimestampValue(timestamp)
			if !checker.IsInvalidValue(value) {
				values = append(values, &moira.MetricValue{Value: value, Timestamp: timestamp})
			}
		}
		triggerMetrics.Main[timeSeries.Name] = values
	}
	for _, timeSeries := range tts.Additional {
		values := make([]*moira.MetricValue, 0)
		for i := 0; i < len(timeSeries.Values); i++ {
			timestamp := timeSeries.StartTime + int64(i)*timeSeries.StepTime
			value := timeSeries.GetTimestampValue(timestamp)
			if !checker.IsInvalidValue(value) {
				values = append(values, &moira.MetricValue{Value: value, Timestamp: timestamp})
			}
		}
		triggerMetrics.Additional[timeSeries.Name] = values
	}
	return &triggerMetrics, nil
}

func deleteTriggerMetrics(dataBase moira.Database, metricName string, triggerID string, removeAllNodataMetrics bool) *api.ErrorResponse {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return api.ErrorInvalidRequest(fmt.Errorf("trigger not found"))
		}
		return api.ErrorInternalServer(err)
	}

	if err = dataBase.AcquireTriggerCheckLock(triggerID, 10); err != nil {
		return api.ErrorInternalServer(err)
	}
	defer dataBase.DeleteTriggerCheckLock(triggerID)

	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return api.ErrorInvalidRequest(fmt.Errorf("trigger check not found"))
		}
		return api.ErrorInternalServer(err)
	}
	if removeAllNodataMetrics {
		for metricName, metricState := range lastCheck.Metrics {
			if metricState.State == checker.NODATA {
				delete(lastCheck.Metrics, metricName)
			}
		}
	} else {
		_, ok := lastCheck.Metrics[metricName]
		if ok {
			delete(lastCheck.Metrics, metricName)
		}
	}
	lastCheck.UpdateScore()
	if err = dataBase.RemovePatternsMetrics(trigger.Patterns); err != nil {
		return api.ErrorInternalServer(err)
	}
	if err = dataBase.SetTriggerLastCheck(triggerID, &lastCheck, trigger.IsRemote); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
