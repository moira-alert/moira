package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/support"
)

const maxTriggerLockAttempts = 30

// UpdateTrigger update trigger data and trigger metrics in last state.
func UpdateTrigger(dataBase moira.Database, trigger *dto.TriggerModel, triggerID string, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	existedTrigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			return nil, api.ErrorNotFound(fmt.Sprintf("trigger with ID = '%s' does not exists", triggerID))
		}
		return nil, api.ErrorInternalServer(err)
	}
	return saveTrigger(dataBase, &existedTrigger, trigger.ToMoiraTrigger(), triggerID, timeSeriesNames)
}

// saveTrigger create or update trigger data and update trigger metrics in last state.
func saveTrigger(dataBase moira.Database, existedTrigger, newTrigger *moira.Trigger, triggerID string, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	if err := dataBase.AcquireTriggerCheckLock(triggerID, maxTriggerLockAttempts); err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	defer dataBase.DeleteTriggerCheckLock(triggerID) //nolint
	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil && !errors.Is(err, database.ErrNil) {
		return nil, api.ErrorInternalServer(err)
	}

	if !errors.Is(err, database.ErrNil) {
		// sometimes we have no time series names but have important information in LastCheck.Metrics (for example maintenance)
		// so on empty timeSeries we will modify LastCheck only if metric evaluation rules changed (targets, expression, etc.)
		if len(timeSeriesNames) != 0 || metricEvaluationRulesChanged(existedTrigger, newTrigger) {
			for metric := range lastCheck.Metrics {
				if _, ok := timeSeriesNames[metric]; !ok {
					lastCheck.RemoveMetricState(metric)
				}
			}
			lastCheck.RemoveMetricsToTargetRelation()
		}
	} else {
		triggerState := moira.StateNODATA
		if newTrigger.TTLState != nil {
			triggerState = newTrigger.TTLState.ToTriggerState()
		}
		lastCheck = moira.CheckData{
			Metrics: make(map[string]moira.MetricState),
			State:   triggerState,
		}
		lastCheck.UpdateScore()
	}

	if err = dataBase.SetTriggerLastCheck(triggerID, &lastCheck, newTrigger.ClusterKey()); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if err = dataBase.SaveTrigger(triggerID, newTrigger); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	resp := dto.SaveTriggerResponse{
		ID:      triggerID,
		Message: "trigger updated",
	}
	return &resp, nil
}

func metricEvaluationRulesChanged(existedTrigger, newTrigger *moira.Trigger) bool {
	if existedTrigger == nil {
		return true
	}

	// maybe number of targets has changed
	if len(existedTrigger.Targets) != len(newTrigger.Targets) {
		return true
	}

	// maybe number one of targets has changed
	for i := range existedTrigger.Targets {
		if existedTrigger.Targets[i] != newTrigger.Targets[i] {
			return true
		}
	}

	// maybe trigger type changed
	if existedTrigger.TriggerType != newTrigger.TriggerType {
		return true
	}

	// maybe warn value changed
	if !equalTwoPointerValues(existedTrigger.WarnValue, newTrigger.WarnValue) {
		return true
	}

	// maybe error value changed
	if !equalTwoPointerValues(existedTrigger.ErrorValue, newTrigger.ErrorValue) {
		return true
	}

	// maybe TTLState changed
	if !equalTwoPointerValues(existedTrigger.TTLState, newTrigger.TTLState) {
		return true
	}

	// maybe expression changed
	if !equalTwoPointerValues(existedTrigger.Expression, newTrigger.Expression) {
		return true
	}

	// maybe trigger source or cluster changed
	if existedTrigger.ClusterKey().String() != newTrigger.ClusterKey().String() {
		return true
	}

	// maybe alone metrics changed
	if len(existedTrigger.AloneMetrics) != len(newTrigger.AloneMetrics) {
		return true
	}

	for targetID := range existedTrigger.AloneMetrics {
		if _, ok := newTrigger.AloneMetrics[targetID]; !ok {
			return true
		}
	}

	return false
}

func equalTwoPointerValues[T comparable](first, second *T) bool {
	if first != nil && second != nil {
		return *first == *second
	}

	return first == nil && second == nil
}

// GetTrigger gets trigger with his throttling - next allowed message time.
func GetTrigger(dataBase moira.Database, triggerID string) (*dto.Trigger, *api.ErrorResponse) {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			return nil, api.ErrorNotFound("trigger not found")
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

// RemoveTrigger deletes trigger by given triggerID.
func RemoveTrigger(database moira.Database, triggerID string) *api.ErrorResponse {
	if err := database.RemoveTrigger(triggerID); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// GetTriggerThrottling gets trigger throttling timestamp.
func GetTriggerThrottling(database moira.Database, triggerID string) (*dto.ThrottlingResponse, *api.ErrorResponse) {
	throttling, _ := database.GetTriggerThrottling(triggerID)
	throttlingUnix := throttling.Unix()
	if throttlingUnix < time.Now().Unix() {
		throttlingUnix = 0
	}
	return &dto.ThrottlingResponse{Throttling: throttlingUnix}, nil
}

// GetTriggerLastCheck gets trigger last check data.
func GetTriggerLastCheck(dataBase moira.Database, triggerID string) (*dto.TriggerCheck, *api.ErrorResponse) {
	lastCheck := &moira.CheckData{}
	var err error

	*lastCheck, err = dataBase.GetTriggerLastCheck(triggerID)
	if err != nil {
		if !errors.Is(err, database.ErrNil) {
			return nil, api.ErrorInternalServer(err)
		}
		lastCheck = nil
	}

	if lastCheck != nil {
		lastCheck.RemoveDeadMetrics()
	}

	triggerCheck := dto.TriggerCheck{
		CheckData: lastCheck,
		TriggerID: triggerID,
	}

	return &triggerCheck, nil
}

// DeleteTriggerThrottling deletes trigger throttling.
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

// SetTriggerMaintenance sets maintenance to metrics and whole trigger.
func SetTriggerMaintenance(database moira.Database, triggerID string, triggerMaintenance dto.TriggerMaintenance, userLogin string, timeCallMaintenance int64) *api.ErrorResponse {
	if err := database.AcquireTriggerCheckLock(triggerID, maxTriggerLockAttempts); err != nil {
		return api.ErrorInternalServer(err)
	}
	defer database.ReleaseTriggerCheckLock(triggerID)
	if err := database.SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, userLogin, timeCallMaintenance); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// GetTriggerDump returns raw trigger from database.
func GetTriggerDump(database moira.Database, logger moira.Logger, triggerID string) (*dto.TriggerDump, *api.ErrorResponse) {
	trigger, err := support.HandlePullTrigger(logger, database, triggerID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	metrics, errMetrics := support.HandlePullTriggerMetrics(logger, database, triggerID)
	if errMetrics != nil {
		return nil, api.ErrorInternalServer(errMetrics)
	}

	lastCheck, errLastCheck := GetTriggerLastCheck(database, triggerID)
	if errLastCheck != nil {
		return nil, errLastCheck
	}

	return &dto.TriggerDump{
		Created:   time.Now().UTC().String(),
		LastCheck: *lastCheck.CheckData,
		Trigger:   *trigger,
		Metrics:   metrics,
	}, nil
}
