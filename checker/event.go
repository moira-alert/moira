package checker

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

var badStateReminder = map[string]int64{
	ERROR:  86400,
	NODATA: 86400,
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
	if !needSend {
		return currentCheck, nil
	}

	if message == nil {
		message = &currentCheck.Message
	}

	eventOldState := lastStateValue
	if lastStateSuppressed {
		eventOldState = lastStateSuppressedValue
	}

	event := moira.NotificationEvent{
		IsTriggerEvent: true,
		TriggerID:      triggerChecker.TriggerID,
		State:          currentStateValue,
		OldState:       eventOldState,
		Timestamp:      timestamp,
		Metric:         triggerChecker.trigger.Name,
		Message:        message,
	}

	currentCheck.EventTimestamp = timestamp
	currentCheck.Suppressed = false

	if triggerChecker.isTriggerSuppressed(&event, timestamp, 0, "") {
		currentCheck.Suppressed = true
		if !lastStateSuppressed {
			currentCheck.SuppressedState = lastStateValue
		}
		return currentCheck, nil
	}

	currentCheck.SuppressedState = ""
	triggerChecker.Logger.Infof("Writing new event: %v", event)
	err := triggerChecker.Database.PushNotificationEvent(&event, true)
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
	if !needSend {
		return currentState, nil
	}

	eventOldState := lastState.State
	if lastState.Suppressed {
		eventOldState = lastState.SuppressedState
	}

	event := moira.NotificationEvent{
		TriggerID: triggerChecker.TriggerID,
		State:     currentState.State,
		OldState:  eventOldState,
		Timestamp: currentState.Timestamp,
		Metric:    metric,
		Message:   message,
		Value:     currentState.Value,
	}

	currentState.EventTimestamp = currentState.Timestamp
	currentState.Suppressed = false

	if triggerChecker.isTriggerSuppressed(&event, currentState.Timestamp, currentState.Maintenance, metric) {
		currentState.Suppressed = true
		if !lastState.Suppressed {
			currentState.SuppressedState = lastState.State
		}
		return currentState, nil
	}

	currentState.SuppressedState = ""
	triggerChecker.Logger.Infof("Writing new event: %v", event)
	err := triggerChecker.Database.PushNotificationEvent(&event, true)
	return currentState, err
}

func (triggerChecker *TriggerChecker) isTriggerSuppressed(event *moira.NotificationEvent, timestamp int64, stateMaintenance int64, metric string) bool {
	if !triggerChecker.trigger.Schedule.IsScheduleAllows(timestamp) {
		triggerChecker.Logger.Debugf("Event %v suppressed due to trigger schedule", event)
		return true
	}
	if stateMaintenance >= timestamp {
		triggerChecker.Logger.Debugf("Event %v suppressed due to metric %s maintenance until %v.", event, metric, time.Unix(stateMaintenance, 0))
		return true
	}
	return false
}

func needSendEvent(currentStateValue string, lastStateValue string, currentStateTimestamp int64, lastStateEventTimestamp int64, isLastCheckSuppressed bool, lastStateSuppressedValue string) (needSend bool, message *string) {
	if !isLastCheckSuppressed && currentStateValue != lastStateValue {
		return true, nil
	}
	if isLastCheckSuppressed && currentStateValue != lastStateSuppressedValue {
		message := "This metric changed its state during maintenance interval."
		return true, &message
	}
	remindInterval, ok := badStateReminder[currentStateValue]
	if ok && needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval) {
		message := fmt.Sprintf("This metric has been in bad state for more than %v hours - please, fix.", remindInterval/3600)
		return true, &message
	}
	return false, nil
}

func needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval int64) bool {
	return currentStateTimestamp-lastStateEventTimestamp >= remindInterval
}
