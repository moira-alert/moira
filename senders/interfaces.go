package senders

// MetricsMarker can be used in senders, that have ability to check
// if the notification was delivered to the recipient.
type MetricsMarker interface {
	// MarkDeliveryFailed should be used than notification was not delivered to recipient or other issues happened.
	MarkDeliveryFailed()
	// MarkDeliveryOK should be used than notification was successfully delivered.
	MarkDeliveryOK()
}
