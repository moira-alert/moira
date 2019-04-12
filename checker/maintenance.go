package checker

import (
	"bytes"
	"time"

	"github.com/moira-alert/moira"
)

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

func getMaintenanceInfoMessage(info moira.MaintenanceInfo) string {
	messageBuffer := bytes.NewBuffer([]byte(""))
	messageBuffer.WriteString("This metric changed its state during maintenance interval.")

	if info.StartUser != nil || info.StartTime != nil {
		messageBuffer.WriteString(" Maintenance was set")
		if info.StartUser != nil {
			messageBuffer.WriteString(" by ")
			messageBuffer.WriteString(*info.StartUser)
		}
		if info.StartTime != nil {
			messageBuffer.WriteString(" at ")
			messageBuffer.WriteString(time.Unix(*info.StartTime, 0).Format(format))
		}
		if info.StopUser != nil || info.StopTime != nil {
			messageBuffer.WriteString(" and removed")
			if info.StopUser != nil && *info.StopUser != *info.StartUser {
				messageBuffer.WriteString(" by ")
				messageBuffer.WriteString(*info.StopUser)
			}
			if info.StopTime != nil {
				messageBuffer.WriteString(" at ")
				messageBuffer.WriteString(time.Unix(*info.StopTime, 0).Format(format))
			}
		}
		messageBuffer.WriteString(".")
	}
	return messageBuffer.String()
}
