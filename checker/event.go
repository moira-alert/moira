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

	// Moira 2.2 compatibility
	if lastStateSuppressed && lastStateSuppressedValue == "" {
		lastStateSuppressedValue = lastStateValue
	}
	currentCheck.SuppressedState = lastStateSuppressedValue

	maintenanceInfo, maintenanceTimestamp := getMaintenanceInfo(lastCheck, nil)
	needSend, message := isStateChanged(currentStateValue, lastStateValue, currentCheckTimestamp, lastCheck.GetEventTimestamp(), lastStateSuppressed, lastStateSuppressedValue, maintenanceTimestamp, maintenanceInfo)
	if !needSend {
		if maintenanceTimestamp < currentCheckTimestamp {
			currentCheck.Suppressed = false
			currentCheck.SuppressedState = ""
		}
		return currentCheck, nil
	}

	if message == nil {
		message = &currentCheck.Message
	}

	currentCheck.EventTimestamp = currentCheckTimestamp

	if triggerChecker.isTriggerSuppressed(currentCheckTimestamp, maintenanceTimestamp) {
		currentCheck.Suppressed = true
		if !lastStateSuppressed {
			currentCheck.SuppressedState = lastStateValue
		}
		return currentCheck, nil
	}

	currentCheck.Suppressed = false
	currentCheck.SuppressedState = ""

	err := triggerChecker.database.PushNotificationEvent(&moira.NotificationEvent{
		IsTriggerEvent: true,
		TriggerID:      triggerChecker.triggerID,
		State:          currentStateValue,
		OldState:       getEventOldState(lastCheck.State, lastCheck.SuppressedState, lastCheck.Suppressed),
		Timestamp:      currentCheckTimestamp,
		Metric:         triggerChecker.trigger.Name,
		Message:        message,
	}, true)
	return currentCheck, err
}

func (triggerChecker *TriggerChecker) compareMetricStates(metric string, currentState moira.MetricState, lastState moira.MetricState) (moira.MetricState, error) {
	// Just set check info
	// TODO: make sure that this logic can be moved to current state initialization
	if lastState.EventTimestamp != 0 {
		currentState.EventTimestamp = lastState.EventTimestamp
	} else {
		currentState.EventTimestamp = currentState.Timestamp
	}

	// Moira 2.2 compatibility
	if lastState.Suppressed && lastState.SuppressedState == "" {
		lastState.SuppressedState = lastState.State
	}
	currentState.SuppressedState = lastState.SuppressedState

	maintenanceInfo, maintenanceTimestamp := getMaintenanceInfo(triggerChecker.lastCheck, &currentState)
	needSend, message := isStateChanged(currentState.State, lastState.State, currentState.Timestamp, lastState.GetEventTimestamp(), lastState.Suppressed, lastState.SuppressedState, maintenanceTimestamp, maintenanceInfo)
	if !needSend {
		if maintenanceTimestamp < currentState.Timestamp {
			currentState.Suppressed = false
			currentState.SuppressedState = ""
		}
		return currentState, nil
	}

	// State was changed. Set event timestamp. Event will be not sent if it is suppressed
	currentState.EventTimestamp = currentState.Timestamp

	if triggerChecker.isTriggerSuppressed(currentState.Timestamp, maintenanceTimestamp) {
		currentState.Suppressed = true
		if !lastState.Suppressed {
			currentState.SuppressedState = lastState.State
		}
		return currentState, nil
	}

	currentState.Suppressed = false
	currentState.SuppressedState = ""

	err := triggerChecker.database.PushNotificationEvent(&moira.NotificationEvent{
		TriggerID: triggerChecker.triggerID,
		State:     currentState.State,
		OldState:  getEventOldState(lastState.State, lastState.SuppressedState, lastState.Suppressed),
		Timestamp: currentState.Timestamp,
		Metric:    metric,
		Message:   message,
		Value:     currentState.Value,
	}, true)
	return currentState, err
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

func getEventOldState(lastCheckState moira.State, lastCheckSuppressedState moira.State, isSuppressed bool) moira.State {
	if isSuppressed {
		return lastCheckSuppressedState
	}
	return lastCheckState
}

func (triggerChecker *TriggerChecker) isTriggerSuppressed(timestamp int64, maintenanceTimestamp int64) bool {
	return !triggerChecker.trigger.Schedule.IsScheduleAllows(timestamp) || maintenanceTimestamp >= timestamp
}

func isStateChanged(currentStateValue moira.State, lastStateValue moira.State, currentStateTimestamp int64, lastStateEventTimestamp int64, isLastCheckSuppressed bool, lastStateSuppressedValue moira.State, maintenance int64, maintenanceInfo moira.MaintenanceInfo) (needSend bool, message *string) {
	if !isLastCheckSuppressed && currentStateValue != lastStateValue {
		return true, nil
	}

	if isLastCheckSuppressed && currentStateValue != lastStateSuppressedValue {
		if currentStateTimestamp > maintenance || currentStateValue != lastStateValue {
			message := getMaintenanceCreateMessage(maintenanceInfo)
			return true, &message
		}
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
