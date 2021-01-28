package notifier

import "github.com/moira-alert/moira"

func getLogWithPackageContext(log *moira.Logger, pkg *NotificationPackage) moira.Logger {
	return (*log).Clone().
		String(moira.LogFieldNameContactID, pkg.Contact.ID).
		String(moira.LogFieldNameTriggerID, pkg.Trigger.ID)
}

func getLogWithEventContext(logger *moira.Logger, event *moira.NotificationEvent) moira.Logger {
	return (*logger).Clone().
		String(moira.LogFieldNameContactID, event.ContactID).
		String(moira.LogFieldNameTriggerID, event.TriggerID).
		String(moira.LogFieldNameSubscriptionID, moira.UseString(event.SubscriptionID))
}
