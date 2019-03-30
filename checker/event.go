package checker

import (
	"bytes"
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

var badStateReminder = map[moira.State]int64{
	moira.StateERROR:  86400,
	moira.StateNODATA: 86400,
}

const (
	format        = "15:04 02.01.2006"
	remindMessage = "This metric has been in bad state for more than %v hours - please, fix."
)

func (triggerChecker *TriggerChecker) compareTriggerStates(currentCheck moira.CheckData) (moira.CheckData, error) {
	lastCheck := triggerChecker.lastCheck

	currentStateValue := currentCheck.State
	lastStateValue := lastCheck.State
	lastStateSuppressed := lastCheck.Suppressed
	lastStateSuppressedValue := lastCheck.SuppressedState
	currentCheckTimestamp := currentCheck.Timestamp

	// TODO: also these fields are put in current check data initialization func, make sure that this logic can be merged with that init logic
	if lastCheck.EventTimestamp != 0 {
		currentCheck.EventTimestamp = lastCheck.EventTimestamp
	} else {
		currentCheck.EventTimestamp = currentCheckTimestamp
	}

	if lastStateSuppressed && lastStateSuppressedValue == "" {
		lastStateSuppressedValue = lastStateValue
	}

	currentCheck.SuppressedState = lastStateSuppressedValue
	maintenanceInfo, maintenanceTimestamp := getMaintenanceInfo(triggerChecker.lastCheck, nil)

	needSend, message := needSendEvent(currentStateValue, lastStateValue, currentCheckTimestamp, lastCheck.GetEventTimestamp(), lastStateSuppressed, lastStateSuppressedValue, maintenanceInfo)
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
		TriggerID:      triggerChecker.triggerID,
		State:          currentStateValue,
		OldState:       eventOldState,
		Timestamp:      currentCheckTimestamp,
		Metric:         triggerChecker.trigger.Name,
		Message:        message,
	}

	currentCheck.EventTimestamp = currentCheckTimestamp
	currentCheck.Suppressed = false

	if triggerChecker.isTriggerSuppressed(currentCheckTimestamp, maintenanceTimestamp) {
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
	// TODO: make sure that this logic can be moved to current state initialization
	if lastState.EventTimestamp != 0 {
		currentState.EventTimestamp = lastState.EventTimestamp
	} else {
		currentState.EventTimestamp = currentState.Timestamp
	}

	if lastState.Suppressed && lastState.SuppressedState == "" {
		lastState.SuppressedState = lastState.State
	}

	currentState.SuppressedState = lastState.SuppressedState
	maintenanceInfo, maintenanceTimestamp := getMaintenanceInfo(triggerChecker.lastCheck, &currentState)

	needSend, message := needSendEvent(currentState.State, lastState.State, currentState.Timestamp, lastState.GetEventTimestamp(), lastState.Suppressed, lastState.SuppressedState, maintenanceInfo)
	if !needSend {
		return currentState, nil
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

	if triggerChecker.isTriggerSuppressed(currentState.Timestamp, maintenanceTimestamp) {
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

func (triggerChecker *TriggerChecker) isTriggerSuppressed(timestamp int64, maintenanceTimestamp int64) bool {
	return !triggerChecker.trigger.Schedule.IsScheduleAllows(timestamp) || maintenanceTimestamp >= timestamp
}

// We must always check triggerMaintenance along with metricMaintenance to avoid cases when metric is not suppressed, but trigger is.
func getMaintenanceInfo(previousTriggerState *moira.CheckData, previousMetricState *moira.MetricState) (moira.MaintenanceInfo, int64) {
	if previousMetricState == nil {
		return previousTriggerState.GetMaintenance(), previousTriggerState.Maintenance
	}
	if previousMetricState.Maintenance >= previousTriggerState.Maintenance {
		return previousMetricState.GetMaintenance(), previousMetricState.Maintenance
	}
	return previousTriggerState.GetMaintenance(), previousTriggerState.Maintenance
}

func needSendEvent(currentStateValue moira.State, lastStateValue moira.State, currentStateTimestamp int64, lastStateEventTimestamp int64, isLastCheckSuppressed bool, lastStateSuppressedValue moira.State, maintenanceInfo moira.MaintenanceInfo) (needSend bool, message *string) {
	if !isLastCheckSuppressed && currentStateValue != lastStateValue {
		return true, nil
	}

	if isLastCheckSuppressed && currentStateValue != lastStateSuppressedValue {
		message := getMaintenanceCreateMessage(maintenanceInfo)
		return true, &message
	}
	remindInterval, ok := badStateReminder[currentStateValue]
	if ok && needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval) {
		message := fmt.Sprintf(remindMessage, remindInterval/3600)
		return true, &message
	}
	return false, nil
}

func needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval int64) bool {
	return currentStateTimestamp-lastStateEventTimestamp >= remindInterval
}

func getMaintenanceCreateMessage(info moira.MaintenanceInfo) string {
	messageBuffer := bytes.NewBuffer([]byte(""))
	messageBuffer.WriteString("This metric changed its state during maintenance interval.")

	if info.StartUser != nil || info.StartTime != nil {
		messageBuffer.WriteString(" Maintenance was set")
		if info.StartUser != nil {
			messageBuffer.WriteString(" by user ")
			messageBuffer.WriteString(*info.StartUser)
		}
		if info.StartTime != nil {
			messageBuffer.WriteString(" at ")
			messageBuffer.WriteString(time.Unix(*info.StartTime, 0).Format(format))
		}
		messageBuffer.WriteString(".")
	}
	if info.StopUser != nil || info.StopTime != nil {
		messageBuffer.WriteString(" Maintenance removed")
		if info.StopUser != nil {
			messageBuffer.WriteString(" by user ")
			messageBuffer.WriteString(*info.StopUser)
		}
		if info.StopTime != nil {
			messageBuffer.WriteString(" at ")
			messageBuffer.WriteString(time.Unix(*info.StopTime, 0).Format(format))
		}
		messageBuffer.WriteString(".")
	}
	return messageBuffer.String()
}
