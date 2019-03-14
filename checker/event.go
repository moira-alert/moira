package checker

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

var badStateReminder = map[moira.State]int64{
	moira.StateERROR:  86400,
	moira.StateNODATA: 86400,
}

func (triggerChecker *TriggerChecker) compareTriggerStates(currentCheck moira.CheckData) (moira.CheckData, error) {
	currentStateValue := currentCheck.State
	lastStateValue := triggerChecker.lastCheck.State
	lastStateSuppressed := triggerChecker.lastCheck.Suppressed
	lastStateSuppressedValue := triggerChecker.lastCheck.SuppressedState
	timestamp := currentCheck.Timestamp

	if triggerChecker.lastCheck.EventTimestamp != 0 {
		currentCheck.EventTimestamp = triggerChecker.lastCheck.EventTimestamp
	} else {
		currentCheck.EventTimestamp = timestamp
	}

	if lastStateSuppressed && lastStateSuppressedValue == "" {
		lastStateSuppressedValue = lastStateValue
	}

	currentCheck.SuppressedState = lastStateSuppressedValue

	needSend, message := needSendEvent(currentStateValue, lastStateValue, timestamp, triggerChecker.lastCheck.GetEventTimestamp(), lastStateSuppressed, lastStateSuppressedValue)
	if needSend != -1 {
		return currentCheck, nil
	}

	if needSend == 0 {
		message = &currentCheck.Message
	}

  if needSend == 1 {
   *message += fmt.Sprintf("Maintenance was set by user %v at %v and expired at %v.",
   	currentCheck.MaintenanceWho.StartMaintenanceUser,
   	time.Unix(*triggerChecker.lastCheck.MaintenanceWho.StartMaintenanceTime, 0),
   	time.Unix(triggerChecker.lastCheck.Maintenance, 0),
   	)
	}

	eventOldState := lastStateValue
	if lastStateSuppressed {
		eventOldState = lastStateSuppressedValue
	}

	event := moira.NotificationEvent{
		IsTriggerEvent: true,
		TriggerID:      triggerChecker.triggerID,
		State:          currentStateValue,
		OldState:       eventOldState,
		Timestamp:      timestamp,
		Metric:         triggerChecker.trigger.Name,
		Message:        message,
	}

	currentCheck.EventTimestamp = timestamp
	currentCheck.Suppressed = false

	if triggerChecker.isTriggerSuppressed(&event, timestamp, 0, currentCheck.Maintenance, "") {
		currentCheck.Suppressed = true
		if !lastStateSuppressed {
			currentCheck.SuppressedState = lastStateValue
		}
		return currentCheck, nil
	}

	currentCheck.SuppressedState = ""
	triggerChecker.logger.Debugf("Writing new event: %v", event)
	err := triggerChecker.database.PushNotificationEvent(&event, true)
	return currentCheck, err
}

func (triggerChecker *TriggerChecker) compareMetricStates(metric string, currentState moira.MetricState, lastState moira.MetricState) (moira.MetricState, error) {
	if lastState.EventTimestamp != 0 {
		currentState.EventTimestamp = lastState.EventTimestamp
	} else {
		currentState.EventTimestamp = currentState.Timestamp
	}

	if lastState.Suppressed && lastState.SuppressedState == "" {
		lastState.SuppressedState = lastState.State
	}

	currentState.SuppressedState = lastState.SuppressedState

	needSend, message := needSendEvent(currentState.State, lastState.State, currentState.Timestamp, lastState.GetEventTimestamp(), lastState.Suppressed, lastState.SuppressedState)
	if needSend != -1 {
		return currentState, nil
	}

	if needSend == 1 {
		*message += fmt.Sprintf("Maintenance was set by user %v at %v and expired at %v.",
			currentState.MaintenanceWho.StartMaintenanceUser,
			time.Unix(*triggerChecker.lastCheck.MaintenanceWho.StartMaintenanceTime, 0),
			time.Unix(triggerChecker.lastCheck.Maintenance, 0),
		)
	}
	eventOldState := lastState.State
	if lastState.Suppressed {
		eventOldState = lastState.SuppressedState
	}

	event := moira.NotificationEvent{
		TriggerID: triggerChecker.triggerID,
		State:     currentState.State,
		OldState:  eventOldState,
		Timestamp: currentState.Timestamp,
		Metric:    metric,
		Message:   message,
		Value:     currentState.Value,
	}

	currentState.EventTimestamp = currentState.Timestamp
	currentState.Suppressed = false

	if triggerChecker.isTriggerSuppressed(&event, currentState.Timestamp, currentState.Maintenance, triggerChecker.lastCheck.Maintenance, metric) {
		currentState.Suppressed = true
		if !lastState.Suppressed {
			currentState.SuppressedState = lastState.State
		}
		return currentState, nil
	}

	currentState.SuppressedState = ""
	triggerChecker.logger.Debugf("Writing new event: %v", event)
	err := triggerChecker.database.PushNotificationEvent(&event, true)
	return currentState, err
}

func (triggerChecker *TriggerChecker) isTriggerSuppressed(event *moira.NotificationEvent, timestamp int64, metricMaintenance int64, triggerMaintenance int64, metric string) bool {
	if !triggerChecker.trigger.Schedule.IsScheduleAllows(timestamp) {
		triggerChecker.logger.Debugf("Event %v suppressed due to trigger schedule", event)
		return true
	}
	// We must always check triggerMaintenance along with metricMaintenance to avoid cases when metric is not suppressed, but trigger is.
	if triggerMaintenance >= timestamp {
		triggerChecker.logger.Debugf("Event %v suppressed due to trigger %s maintenance until %v. Maintenance was set by user %v at %v.", event, triggerChecker.trigger.ID, time.Unix(triggerMaintenance, 0), triggerChecker.lastCheck.MaintenanceWho.StartMaintenanceUser, time.Unix(*triggerChecker.lastCheck.MaintenanceWho.StartMaintenanceTime, 0))
		return true
	}
	if metricMaintenance >= timestamp {
		triggerChecker.logger.Debugf("Event %v suppressed due to metric %s maintenance until %v. Maintenance was set by user %v at %v.", event, metric, time.Unix(metricMaintenance, 0),  triggerChecker.lastCheck.MaintenanceWho.StartMaintenanceUser, time.Unix(*triggerChecker.lastCheck.MaintenanceWho.StartMaintenanceTime, 0))
		return true
	}
	return false
}

func needSendEvent(currentStateValue moira.State, lastStateValue moira.State, currentStateTimestamp int64, lastStateEventTimestamp int64, isLastCheckSuppressed bool, lastStateSuppressedValue moira.State) (needSend int, message *string) {
	if !isLastCheckSuppressed && currentStateValue != lastStateValue {
		return 0, nil
	}
	if isLastCheckSuppressed && currentStateValue != lastStateSuppressedValue {
		message := "This metric changed its state during maintenance interval."
		return 1, &message
	}
	remindInterval, ok := badStateReminder[currentStateValue]
	if ok && needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval) {
		message := fmt.Sprintf("This metric has been in bad state for more than %v hours - please, fix.", remindInterval/3600)
		return 2, &message
	}
	return -1, nil
}

func needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval int64) bool {
	return currentStateTimestamp-lastStateEventTimestamp >= remindInterval
}
