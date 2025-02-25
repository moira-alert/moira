package datatypes

// DeliveryCheckerDatabase is used by senders that can track if the notification was delivered.
type DeliveryCheckerDatabase interface {
	// AddNotificationsToCheckDelivery must be used than you want for some alerts to be checked if they have been delivered.
	AddNotificationsToCheckDelivery(contactType string, timestamp int64, data string) error
	// GetNotificationsToCheckDelivery must be used to get alerts that need to be checked.
	GetNotificationsToCheckDelivery(contactType string, from string, to string) ([]string, error)
	// RemoveNotificationsToCheckDelivery removes already checked alerts.
	RemoveNotificationsToCheckDelivery(contactType string, from string, to string) (int64, error)
}
