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

const (
	// DeliveryStateOK means that alert was successfully delivered.
	DeliveryStateOK = "OK"
	// DeliveryStatePending means that alert has not yet been delivered.
	DeliveryStatePending = "PENDING"
	// DeliveryStateFailed means that alert was not delivered.
	DeliveryStateFailed = "FAILED"
	// DeliveryStateException means that error occurred during checking (not by user fault). For example, connection problems, etc.
	DeliveryStateException = "EXCEPTION"
	// DeliveryStateUserException means that error occurred during checking (by user fault). For example, bad template in config.
	DeliveryStateUserException = "USER_EXCEPTION"
)
