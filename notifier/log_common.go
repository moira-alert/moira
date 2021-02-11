package notifier

import "github.com/moira-alert/moira"

func getLogWithPackageContext(log *moira.Logger, pkg *NotificationPackage) moira.Logger {
	return (*log).Clone().
		String(moira.LogFieldNameContactID, pkg.Contact.ID).
		String(moira.LogFieldNameContactType, pkg.Contact.Type).
		String(moira.LogFieldNameContactValue, pkg.Contact.Value).
		String(moira.LogFieldNameTriggerID, pkg.Trigger.ID).
		String(moira.LogFieldNameTriggerName, pkg.Trigger.Name)
}
