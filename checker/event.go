package checker

import (
	"github.com/moira-alert/moira"
)

var badStateReminder = map[moira.State]int64{
	moira.StateERROR:  86400,
	moira.StateNODATA: 86400,
}

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
	eventInfo, needSend := isStateChanged(currentStateValue, lastStateValue, currentCheckTimestamp, lastCheck.GetEventTimestamp(), lastStateSuppressed, lastStateSuppressedValue, maintenanceInfo)
	if !needSend {
		if maintenanceTimestamp < currentCheckTimestamp {
			currentCheck.Suppressed = false
			currentCheck.SuppressedState = ""
		}
		return currentCheck, nil
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
		IsTriggerEvent:   true,
		TriggerID:        triggerChecker.triggerID,
		State:            currentStateValue,
		OldState:         getEventOldState(lastCheck.State, lastCheck.SuppressedState, lastCheck.Suppressed),
		Timestamp:        currentCheckTimestamp,
		Metric:           triggerChecker.trigger.Name,
		MessageEventInfo: eventInfo,
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
	eventInfo, needSend := isStateChanged(currentState.State, lastState.State, currentState.Timestamp, lastState.GetEventTimestamp(), lastState.Suppressed, lastState.SuppressedState, maintenanceInfo)
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
		TriggerID:        triggerChecker.triggerID,
		State:            currentState.State,
		OldState:         getEventOldState(lastState.State, lastState.SuppressedState, lastState.Suppressed),
		Timestamp:        currentState.Timestamp,
		Metric:           metric,
		MessageEventInfo: eventInfo,
		Values:           currentState.Values,
	}, true)
	return currentState, err
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

func isStateChanged(currentStateValue moira.State, lastStateValue moira.State, currentStateTimestamp int64, lastStateEventTimestamp int64, isLastCheckSuppressed bool, lastStateSuppressedValue moira.State, maintenanceInfo moira.MaintenanceInfo) (*moira.EventInfo, bool) {
	if !isLastCheckSuppressed && currentStateValue != lastStateValue {
		return nil, true
	}

	if isLastCheckSuppressed && currentStateValue != lastStateSuppressedValue {
		return &moira.EventInfo{Maintenance: &maintenanceInfo}, true
	}

	remindInterval, ok := badStateReminder[currentStateValue]
	if ok && needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval) {
		interval := remindInterval / 3600
		return &moira.EventInfo{Interval: &interval}, true
	}
	return nil, false
}

func needRemindAgain(currentStateTimestamp, lastStateEventTimestamp, remindInterval int64) bool {
	return currentStateTimestamp-lastStateEventTimestamp >= remindInterval
}

// We must always check triggerMaintenance along with metricMaintenance to avoid cases when metric is not suppressed, but trigger is.
func getMaintenanceInfo(triggerState moira.MaintenanceCheck, metricState moira.MaintenanceCheck) (moira.MaintenanceInfo, int64) {
	if metricState == nil {
		return triggerState.GetMaintenance()
	}
	if triggerState == nil {
		return metricState.GetMaintenance()
	}
	triggerTS := getCompareTimestamp(triggerState)
	metricTS := getCompareTimestamp(metricState)

	if metricTS >= triggerTS {
		return metricState.GetMaintenance()
	}
	return triggerState.GetMaintenance()
}

func getCompareTimestamp(mainCheck moira.MaintenanceCheck) int64 {
	mainInfo, mainTS := mainCheck.GetMaintenance()
	if mainInfo.StopTime == nil {
		return mainTS
	}
	removeTime := *mainInfo.StopTime
	if removeTime > mainTS {
		return removeTime
	}
	return mainTS
}
