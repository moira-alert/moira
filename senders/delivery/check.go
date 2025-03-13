package delivery

// CheckAction represents action that is performed to check the delivery of notifications.
type CheckAction interface {
	// CheckNotificationsDelivery should check if notifications delivery state and return
	// data to schedule again.
	CheckNotificationsDelivery(fetchedDeliveryChecks []string, counter *TypesCounter) []string
}

// TypesCounter contains counters for different types of delivery statuses.
type TypesCounter struct {
	// DeliveryOK is the number of notifications successfully delivered.
	DeliveryOK int64
	// DeliveryFailed is the number of notifications definitely not delivered.
	DeliveryFailed int64
	// DeliveryChecksStopped is the number of notifications for which delivery checks have been stopped.
	DeliveryChecksStopped int64
}
