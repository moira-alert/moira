package controller

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
	"time"
)

func GetTrigger(database moira.Database, triggerId string) (*dto.Trigger, *dto.ErrorResponse) {
	trigger, err := database.GetTrigger(triggerId)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}
	if trigger == nil {
		return nil, dto.ErrorNotFound
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

func GetTriggerThrottling(database moira.Database, triggerId string) (*dto.ThrottlingResponse, *dto.ErrorResponse) {
	throttling, _ := database.GetTriggerThrottlingTimestamps(triggerId)
	throttlingUnix := throttling.Unix()
	if throttlingUnix < time.Now().Unix() {
		throttlingUnix = 0
	}
	return &dto.ThrottlingResponse{Throttling: throttlingUnix}, nil
}

func GetTriggerState(database moira.Database, triggerId string) (*dto.TriggerCheck, *dto.ErrorResponse) {
	lastCheck, err := database.GetTriggerLastCheck(triggerId)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	triggerCheck := dto.TriggerCheck{
		CheckData: lastCheck,
		TriggerId: triggerId,
	}

	return &triggerCheck, nil
}

func DeleteTriggerThrottling(database moira.Database, triggerId string) *dto.ErrorResponse {
	if err := database.DeleteTriggerThrottling(triggerId); err != nil {
		return dto.ErrorInternalServer(err)
	}
	return nil
}

func DeleteTriggerMetric(database moira.Database, metricName string, triggerId string) *dto.ErrorResponse {
	//todo
	trigger, err := database.GetTrigger(triggerId)
	if err != nil {
		return dto.ErrorInternalServer(err)
	}
	if trigger == nil {
		return dto.ErrorInvalidRequest(fmt.Errorf("Trigger check not found"))
	}
	return nil
}

func SetMetricsMaintenance(database moira.Database, triggerId string, metricsMaintenance *dto.MetricsMaintenance) *dto.ErrorResponse {
	if err := database.SetTriggerMetricsMaintenance(triggerId, map[string]int64(*metricsMaintenance)); err != nil {
		return dto.ErrorInternalServer(err)
	}
	return nil
}
