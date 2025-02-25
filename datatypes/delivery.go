package datatypes

// DeliveryCheckerDatabase is used by senders that can track if the notification was delivered.
type DeliveryCheckerDatabase interface {
	// AddDeliveryChecksData must be used to store data need for performing delivery checks.
	AddDeliveryChecksData(contactType string, timestamp int64, data string) error
	// GetDeliveryChecksData must be used to get data need for performing delivery checks.
	GetDeliveryChecksData(contactType string, from string, to string) ([]string, error)
	// RemoveDeliveryChecksData must remove already used data for performing delivery checks.
	RemoveDeliveryChecksData(contactType string, from string, to string) (int64, error)
}
