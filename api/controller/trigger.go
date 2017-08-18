package controller

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/moira-alert/moira-alert/checker"
	"time"
)

func SaveTrigger(database moira.Database, trigger *moira.Trigger, triggerId string, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	database.AcquireTriggerCheckLock(triggerId, 10)
	lastCheck, err := database.GetTriggerLastCheck(triggerId)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if lastCheck != nil {
		for metric, _ := range lastCheck.Metrics {
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

	if err = database.SetTriggerLastCheck(triggerId, lastCheck); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if database.DeleteTriggerCheckLock(triggerId); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if database.SaveTrigger(triggerId, trigger); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	resp := dto.SaveTriggerResponse{
		Id:      triggerId,
		Message: "trigger updated",
	}
	return &resp, nil
}

func GetTrigger(database moira.Database, triggerId string) (*dto.Trigger, *api.ErrorResponse) {
	trigger, err := database.GetTrigger(triggerId)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	if trigger == nil {
		return nil, api.ErrorNotFound("Trigger not found")
	}
	throttling, _ := database.GetTriggerThrottlingTimestamps(triggerId)
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

func DeleteTrigger(database moira.Database, triggerId string) *api.ErrorResponse {
	if err := database.DeleteTrigger(triggerId); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

func GetTriggerThrottling(database moira.Database, triggerId string) (*dto.ThrottlingResponse, *api.ErrorResponse) {
	throttling, _ := database.GetTriggerThrottlingTimestamps(triggerId)
	throttlingUnix := throttling.Unix()
	if throttlingUnix < time.Now().Unix() {
		throttlingUnix = 0
	}
	return &dto.ThrottlingResponse{Throttling: throttlingUnix}, nil
}

func GetTriggerState(database moira.Database, triggerId string) (*dto.TriggerCheck, *api.ErrorResponse) {
	lastCheck, err := database.GetTriggerLastCheck(triggerId)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	triggerCheck := dto.TriggerCheck{
		CheckData: lastCheck,
		TriggerId: triggerId,
	}

	return &triggerCheck, nil
}

func DeleteTriggerThrottling(database moira.Database, triggerId string) *api.ErrorResponse {
	if err := database.DeleteTriggerThrottling(triggerId); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

func DeleteTriggerMetric(database moira.Database, metricName string, triggerId string) *api.ErrorResponse {
	trigger, err := database.GetTrigger(triggerId)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	if trigger == nil {
		return api.ErrorInvalidRequest(fmt.Errorf("Trigger not found"))
	}

	if err = database.AcquireTriggerCheckLock(triggerId, 10); err != nil {
		return api.ErrorInternalServer(err)
	}
	defer database.DeleteTriggerCheckLock(triggerId)

	lastCheck, err := database.GetTriggerLastCheck(triggerId)
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
	database.SetTriggerLastCheck(triggerId, lastCheck)
	return nil
}

func SetMetricsMaintenance(database moira.Database, triggerId string, metricsMaintenance *dto.MetricsMaintenance) *api.ErrorResponse {
	if err := database.SetTriggerMetricsMaintenance(triggerId, map[string]int64(*metricsMaintenance)); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
