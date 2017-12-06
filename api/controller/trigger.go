package controller

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/target"
)

// UpdateTrigger update trigger data and trigger metrics in last state
func UpdateTrigger(dataBase moira.Database, trigger *dto.TriggerModel, triggerID string, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	_, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil, api.ErrorNotFound(fmt.Sprintf("Trigger with ID = '%s' does not exists", triggerID))
		}
		return nil, api.ErrorInternalServer(err)
	}
	return saveTrigger(dataBase, trigger.ToMoiraTrigger(), triggerID, timeSeriesNames)
}

// saveTrigger create or update trigger data and update trigger metrics in last state
func saveTrigger(dataBase moira.Database, trigger *moira.Trigger, triggerID string, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	if err := dataBase.AcquireTriggerCheckLock(triggerID, 10); err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	defer dataBase.DeleteTriggerCheckLock(triggerID)
	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil && err != database.ErrNil {
		return nil, api.ErrorInternalServer(err)
	}

	if err != database.ErrNil {
		for metric := range lastCheck.Metrics {
			if _, ok := timeSeriesNames[metric]; !ok {
				delete(lastCheck.Metrics, metric)
			}
		}
	} else {
		lastCheck = moira.CheckData{
			Metrics: make(map[string]moira.MetricState),
			State:   checker.NODATA,
		}
		lastCheck.UpdateScore()
	}

	if err = dataBase.SetTriggerLastCheck(triggerID, &lastCheck); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if err = dataBase.SaveTrigger(triggerID, trigger); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	resp := dto.SaveTriggerResponse{
		ID:      triggerID,
		Message: "trigger updated",
	}
	return &resp, nil
}

// GetTrigger gets trigger with his throttling - next allowed message time
func GetTrigger(dataBase moira.Database, triggerID string) (*dto.Trigger, *api.ErrorResponse) {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil, api.ErrorNotFound("Trigger not found")
		}
		return nil, api.ErrorInternalServer(err)
	}
	throttling, _ := dataBase.GetTriggerThrottling(triggerID)
	throttlingUnix := throttling.Unix()

	if throttlingUnix < time.Now().Unix() {
		throttlingUnix = 0
	}

	triggerResponse := dto.Trigger{
		TriggerModel: dto.CreateTriggerModel(&trigger),
		Throttling:   throttlingUnix,
	}

	return &triggerResponse, nil
}

// RemoveTrigger deletes trigger by given triggerID
func RemoveTrigger(database moira.Database, triggerID string) *api.ErrorResponse {
	if err := database.RemoveTrigger(triggerID); err != nil {
		return api.ErrorInternalServer(err)
	}
	if err := database.RemoveTriggerLastCheck(triggerID); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// GetTriggerThrottling gets trigger throttling timestamp
func GetTriggerThrottling(database moira.Database, triggerID string) (*dto.ThrottlingResponse, *api.ErrorResponse) {
	throttling, _ := database.GetTriggerThrottling(triggerID)
	throttlingUnix := throttling.Unix()
	if throttlingUnix < time.Now().Unix() {
		throttlingUnix = 0
	}
	return &dto.ThrottlingResponse{Throttling: throttlingUnix}, nil
}

// GetTriggerLastCheck gets trigger last check data
func GetTriggerLastCheck(dataBase moira.Database, triggerID string) (*dto.TriggerCheck, *api.ErrorResponse) {
	lastCheck := &moira.CheckData{}
	var err error

	*lastCheck, err = dataBase.GetTriggerLastCheck(triggerID)
	if err != nil {
		if err != database.ErrNil {
			return nil, api.ErrorInternalServer(err)
		}
		lastCheck = nil
	}

	triggerCheck := dto.TriggerCheck{
		CheckData: lastCheck,
		TriggerID: triggerID,
	}

	return &triggerCheck, nil
}

// DeleteTriggerThrottling deletes trigger throttling
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

// DeleteTriggerMetric deletes metric from last check and all trigger patterns metrics
func DeleteTriggerMetric(dataBase moira.Database, metricName string, triggerID string) *api.ErrorResponse {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return api.ErrorInvalidRequest(fmt.Errorf("Trigger not found"))
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
			return api.ErrorInvalidRequest(fmt.Errorf("Trigger check not found"))
		}
		return api.ErrorInternalServer(err)
	}
	_, ok := lastCheck.Metrics[metricName]
	if ok {
		delete(lastCheck.Metrics, metricName)
		lastCheck.UpdateScore()
	}
	if err = dataBase.RemovePatternsMetrics(trigger.Patterns); err != nil {
		return api.ErrorInternalServer(err)
	}
	if err = dataBase.SetTriggerLastCheck(triggerID, &lastCheck); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// SetMetricsMaintenance sets metrics maintenance for current trigger
func SetMetricsMaintenance(database moira.Database, triggerID string, metricsMaintenance dto.MetricsMaintenance) *api.ErrorResponse {
	if err := database.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64(metricsMaintenance)); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// GetTriggerMetrics gets all trigger metrics values, default values from: now - 10min, to: now
func GetTriggerMetrics(dataBase moira.Database, from, to int64, triggerID string) (dto.TriggerMetrics, *api.ErrorResponse) {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("Trigger not found"))
		}
		return nil, api.ErrorInternalServer(err)
	}

	triggerMetrics := make(map[string][]moira.MetricValue)
	isSimpleTrigger := trigger.IsSimple()
	for _, tar := range trigger.Targets {
		result, err := target.EvaluateTarget(dataBase, tar, from, to, isSimpleTrigger)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		for _, timeSeries := range result.TimeSeries {
			values := make([]moira.MetricValue, 0)
			for i := 0; i < len(timeSeries.Values); i++ {
				timestamp := int64(timeSeries.StartTime + int32(i)*timeSeries.StepTime)
				value := timeSeries.GetTimestampValue(timestamp)
				if checker.IsInvalidValue(value) {
					values = append(values, moira.MetricValue{Value: value, Timestamp: timestamp})
				}
			}
			triggerMetrics[timeSeries.Name] = values
		}
	}
	return triggerMetrics, nil
}
