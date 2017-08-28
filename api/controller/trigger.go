package controller

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/moira-alert/moira-alert/checker"
	"github.com/moira-alert/moira-alert/target"
	"math"
	"time"
)

//SaveTrigger create or update trigger data and update trigger metrics in last state
func SaveTrigger(database moira.Database, trigger *moira.Trigger, triggerID string, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	if err := database.AcquireTriggerCheckLock(triggerID, 10); err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	defer database.DeleteTriggerCheckLock(triggerID)
	lastCheck, err := database.GetTriggerLastCheck(triggerID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if lastCheck != nil {
		for metric := range lastCheck.Metrics {
			if _, ok := timeSeriesNames[metric]; !ok {
				delete(lastCheck.Metrics, metric)
			}
		}
	} else {
		lastCheck = &moira.CheckData{
			Metrics: make(map[string]moira.MetricState),
			Score:   0,
			State:   checker.NODATA,
		}
	}

	if err = database.SetTriggerLastCheck(triggerID, lastCheck); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if err = database.SaveTrigger(triggerID, trigger); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	resp := dto.SaveTriggerResponse{
		ID:      triggerID,
		Message: "trigger updated",
	}
	return &resp, nil
}

//GetTrigger gets trigger with his throttling - next allowed message time
func GetTrigger(database moira.Database, triggerID string) (*dto.Trigger, *api.ErrorResponse) {
	trigger, err := database.GetTrigger(triggerID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	if trigger == nil {
		return nil, api.ErrorNotFound("Trigger not found")
	}
	throttling, _ := database.GetTriggerThrottlingTimestamps(triggerID)
	throttlingUnix := throttling.Unix()

	if throttlingUnix < time.Now().Unix() {
		throttlingUnix = 0
	}

	triggerResponse := dto.Trigger{
		Trigger:    *trigger,
		Throttling: throttlingUnix,
	}

	return &triggerResponse, nil
}

//DeleteTrigger deletes triggers
func DeleteTrigger(database moira.Database, triggerID string) *api.ErrorResponse {
	if err := database.DeleteTrigger(triggerID); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

//GetTriggerThrottling gets trigger throttling timestamp
func GetTriggerThrottling(database moira.Database, triggerID string) (*dto.ThrottlingResponse, *api.ErrorResponse) {
	throttling, _ := database.GetTriggerThrottlingTimestamps(triggerID)
	throttlingUnix := throttling.Unix()
	if throttlingUnix < time.Now().Unix() {
		throttlingUnix = 0
	}
	return &dto.ThrottlingResponse{Throttling: throttlingUnix}, nil
}

//GetTriggerLastCheck gets trigger last check data
func GetTriggerLastCheck(database moira.Database, triggerID string) (*dto.TriggerCheck, *api.ErrorResponse) {
	lastCheck, err := database.GetTriggerLastCheck(triggerID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	triggerCheck := dto.TriggerCheck{
		CheckData: lastCheck,
		TriggerID: triggerID,
	}

	return &triggerCheck, nil
}

//DeleteTriggerThrottling deletes trigger throttling
func DeleteTriggerThrottling(database moira.Database, triggerID string) *api.ErrorResponse {
	if err := database.DeleteTriggerThrottling(triggerID); err != nil {
		return api.ErrorInternalServer(err)
	}

	now := time.Now().Unix()
	notifications, _, err := database.GetNotifications(0, -1)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	notificationsForRewrite := make([]*moira.ScheduledNotification, 0)
	for _, notification := range notifications {
		if notification != nil && notification.Event.TriggerID == triggerID {
			notificationsForRewrite = append(notificationsForRewrite, notification)
		}
	}
	if err = database.AddNotifications(notificationsForRewrite, now); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

//DeleteTriggerMetric deletes metric from last check and all trigger patterns metrics
func DeleteTriggerMetric(database moira.Database, metricName string, triggerID string) *api.ErrorResponse {
	trigger, err := database.GetTrigger(triggerID)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	if trigger == nil {
		return api.ErrorInvalidRequest(fmt.Errorf("Trigger not found"))
	}

	if err = database.AcquireTriggerCheckLock(triggerID, 10); err != nil {
		return api.ErrorInternalServer(err)
	}
	defer database.DeleteTriggerCheckLock(triggerID)

	lastCheck, err := database.GetTriggerLastCheck(triggerID)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	if lastCheck == nil {
		return api.ErrorInvalidRequest(fmt.Errorf("Trigger check not found"))
	}
	_, ok := lastCheck.Metrics[metricName]
	if ok {
		delete(lastCheck.Metrics, metricName)
	}
	if err = database.RemovePatternsMetrics(trigger.Patterns); err != nil {
		return api.ErrorInternalServer(err)
	}
	if err = database.SetTriggerLastCheck(triggerID, lastCheck); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

//SetMetricsMaintenance sets metrics maintenance for current trigger
func SetMetricsMaintenance(database moira.Database, triggerID string, metricsMaintenance dto.MetricsMaintenance) *api.ErrorResponse {
	if err := database.SetTriggerMetricsMaintenance(triggerID, map[string]int64(metricsMaintenance)); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

//GetTriggerMetrics gets all trigger metrics values, default values from: now - 10min, to: now
func GetTriggerMetrics(database moira.Database, from, to int64, triggerID string) (dto.TriggerMetrics, *api.ErrorResponse) {
	trigger, err := database.GetTrigger(triggerID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	if trigger == nil {
		return nil, api.ErrorInvalidRequest(fmt.Errorf("Trigger not found"))
	}

	triggerMetrics := make(map[string][]moira.MetricValue)
	for _, tar := range trigger.Targets {
		result, err := target.EvaluateTarget(database, tar, from, to, true)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		for _, timeSeries := range result.TimeSeries {
			values := make([]moira.MetricValue, 0)
			for i := 0; i < len(timeSeries.Values); i++ {
				timestamp := int64(timeSeries.StartTime + int32(i)*timeSeries.StepTime)
				value := timeSeries.GetTimestampValue(timestamp)
				if !math.IsNaN(value) {
					values = append(values, moira.MetricValue{Value: value, Timestamp: timestamp})
				}
			}
			triggerMetrics[timeSeries.Name] = values
		}
	}
	return triggerMetrics, nil
}
