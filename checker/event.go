package checker

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"time"
)

var badStateReminder = map[string]int64{
	ERROR:  86400,
	NODATA: 86400,
}

func (triggerChecker *TriggerChecker) compareChecks(currentCheck moira.CheckData) (moira.CheckData, error) {
	currentStateValue := currentCheck.State
	lastStateValue := triggerChecker.lastCheck.State
	timestamp := currentCheck.Timestamp

	if currentCheck.EventTimestamp == 0 {
		currentCheck.EventTimestamp = timestamp
	}

	needSend, message := needSendEvent(currentStateValue, lastStateValue, timestamp, triggerChecker.lastCheck.GetEventTimestamp(), triggerChecker.lastCheck.Suppressed)
	if !needSend {
		return currentCheck, nil
	}

	event := moira.NotificationEvent{
		TriggerID: triggerChecker.TriggerID,
		State:     currentStateValue,
		OldState:  lastStateValue,
		Timestamp: timestamp,
		Metric:    "",
		Message:   message,
	}

	currentCheck.EventTimestamp = timestamp
	currentCheck.Suppressed = false

	if triggerChecker.isTriggerSuppressed(&event, timestamp, 0, "") {
		currentCheck.Suppressed = true
		return currentCheck, nil
	}
	triggerChecker.Logger.Infof("Writing new event: %v", event)
	err := triggerChecker.Database.PushEvent(&event, true)
	return currentCheck, err
}

func (triggerChecker *TriggerChecker) compareStates(metric string, currentState moira.MetricState, lastState moira.MetricState) (moira.MetricState, error) {
	if lastState.EventTimestamp != 0 {
		currentState.EventTimestamp = lastState.EventTimestamp
	} else {
		currentState.EventTimestamp = currentState.Timestamp
	}

	needSend, message := needSendEvent(currentState.State, lastState.State, currentState.Timestamp, lastState.GetEventTimestamp(), lastState.Suppressed)
	if !needSend {
		return currentState, nil
	}

	event := moira.NotificationEvent{
		TriggerID: triggerChecker.TriggerID,
		State:     currentState.State,
		OldState:  lastState.State,
		Timestamp: currentState.Timestamp,
		Metric:    metric,
		Message:   message,
		Value:     currentState.Value,
	}

	currentState.EventTimestamp = currentState.Timestamp
	currentState.Suppressed = false

	if triggerChecker.isTriggerSuppressed(&event, currentState.Timestamp, currentState.Maintenance, metric) {
		currentState.Suppressed = true
		return currentState, nil
	}
	triggerChecker.Logger.Infof("Writing new event: %v", event)
	err := triggerChecker.Database.PushEvent(&event, true)
	return currentState, err
}

func (triggerChecker *TriggerChecker) isTriggerSuppressed(event *moira.NotificationEvent, timestamp int64, stateMaintenance int64, metric string) bool {
	if !triggerChecker.trigger.Schedule.IsScheduleAllows(timestamp) {
		triggerChecker.Logger.Infof("Event %v suppressed due to trigger schedule", event)
		return true
	}
	if stateMaintenance >= timestamp {
		triggerChecker.Logger.Infof("Event %v suppressed due to metric %s maintenance until %v.", event, metric, time.Unix(stateMaintenance, 0))
		return true
	}
	return false
}

func needSendEvent(currentStateValue string, lastStateValue string, currentStateTimestamp int64, lastStateEventTimestamp int64, isLastStateSuppressed bool) (bool, *string) {
	if currentStateValue != lastStateValue {
		return true, nil
	}
	remindInterval, ok := badStateReminder[currentStateValue]
	if ok && needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval) {
		message := fmt.Sprintf("This metric has been in bad state for more than %v hours - please, fix.", remindInterval/3600)
		return true, &message
	}
	if !isLastStateSuppressed || currentStateValue == OK {
		return false, nil
	}
	return true, nil
}

func needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval int64) bool {
	return currentStateTimestamp-lastStateEventTimestamp >= remindInterval
}
