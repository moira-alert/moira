package delivery

import "github.com/moira-alert/moira"

// CheckAction represents action that is performed to check the delivery of notifications.
type CheckAction interface {
	// CheckNotificationsDelivery should check if notifications delivery state and return
	// data to schedule again.
	CheckNotificationsDelivery(fetchedDeliveryChecks []string) ([]string, moira.DeliveryTypesCounter)
}
