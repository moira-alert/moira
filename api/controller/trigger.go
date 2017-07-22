package controller

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
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

	triggerResponse := dto.Trigger{
		Trigger:    *trigger,
		Throttling: throttling.Unix(),
	}

	return &triggerResponse, nil
}

func GetTriggerState(database moira.Database, triggerId string) (*dto.TriggerCheck, *dto.ErrorResponse) {
	lastCheck, err := database.GetTriggerLastCheck(triggerId)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	triggerCheck := dto.TriggerCheck{
		CheckData: *lastCheck,
		TriggerId: triggerId,
	}

	return &triggerCheck, nil
}

func DeleteTriggerMetric(database moira.Database, metricName string, triggerId string) *dto.ErrorResponse {
	trigger, err := database.GetTrigger(triggerId)
	if err != nil {
		return dto.ErrorInternalServer(err)
	}
	if trigger == nil {
		return dto.ErrorInvalidRequest(fmt.Errorf("Trigger check not found"))
	}
	return nil
}
