package delivery

import "github.com/moira-alert/moira"

// LogFieldPrefix is the recommended prefix for log fields, written to log, when performing delivery checks.
const LogFieldPrefix = "delivery.check."

// NotificationDeliveryChecker represents action that is performed to check the delivery of notifications.
type NotificationDeliveryChecker interface {
	// CheckNotificationsDelivery should check notifications delivery state and return
	// data to schedule again.
	CheckNotificationsDelivery(fetchedDeliveryChecks []string) ([]string, moira.DeliveryTypesCounter)
}
