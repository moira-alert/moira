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

func (triggerChecker *TriggerChecker) compareStates(metric string, currentState *moira.MetricState, lastState *moira.MetricState) error {
	currentStateValue := currentState.State
	lastStateValue := lastState.State

	if currentState.EventTimestamp == 0 {
		currentState.EventTimestamp = currentState.Timestamp
	}

	needSend, message := needSendEvent(currentState, lastState)
	if !needSend {
		return nil
	}

	event := moira.EventData{
		TriggerID: triggerChecker.TriggerId,
		State:     currentStateValue,
		OldState:  lastStateValue,
		Timestamp: currentState.Timestamp,
		Metric:    metric,
		Message:   message,
	}

	currentState.EventTimestamp = currentState.Timestamp
	lastState.EventTimestamp = currentState.Timestamp
	currentState.Suppressed = false
	lastState.Suppressed = false
	if !triggerChecker.trigger.Schedule.IsScheduleAllows(currentState.Timestamp) {
		currentState.Suppressed = true
		triggerChecker.Logger.Infof("Event %v suppressed due to trigger schedule", event)
		return nil
	}
	if triggerChecker.maintenance >= currentState.Timestamp {
		currentState.Suppressed = true
		triggerChecker.Logger.Infof("Event %v suppressed due to maintenance until %v.", event, time.Unix(triggerChecker.maintenance, 0))
		return nil
	}
	stateMaintenance := currentState.Maintenance
	if stateMaintenance >= currentState.Timestamp {
		currentState.Suppressed = true
		triggerChecker.Logger.Infof("Event %v suppressed due to metric %s maintenance until %v.", event, metric, time.Unix(stateMaintenance, 0))
		return nil
	}
	triggerChecker.Logger.Infof("Writing new event: %v", event)
	triggerChecker.Database.PushEvent(&event, false)
	return nil
}

func needSendEvent(currentState *moira.MetricState, lastState *moira.MetricState) (bool, *string) {
	if currentState.State != lastState.State {
		return true, nil
	}
	remindInterval, ok := badStateReminder[currentState.State]
	if ok && needRemindAgain(currentState.Timestamp, lastState.GetEventTimestamp(), remindInterval) {
		message := fmt.Sprintf("This metric has been in bad state for more than %v hours - please, fix.", remindInterval/3600)
		return true, &message
	}
	if !lastState.Suppressed || currentState.State == OK {
		return false, nil
	}
	return true, nil
}

func needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval int64) bool {
	return currentStateTimestamp-lastStateEventTimestamp >= remindInterval
}
