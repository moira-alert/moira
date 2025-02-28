package metrics

// SenderMetrics should be used for sender which can understand if the notification was delivered or not.
type SenderMetrics struct {
	ContactDeliveryNotificationOK           Meter
	ContactDeliveryNotificationFailed       Meter
	ContactDeliveryNotificationCheckStopped Meter
}

// ConfigureSenderMetrics configures SenderMetrics using NotifierMetrics with given graphiteIdent for senderContactType.
func ConfigureSenderMetrics(notifierMetrics *NotifierMetrics, graphiteIdent string, senderContactType string) *SenderMetrics {
	return &SenderMetrics{
		ContactDeliveryNotificationOK: notifierMetrics.ContactsDeliveryNotificationsOK.
			RegisterMeter(senderContactType, graphiteIdent, "delivery_ok"),
		ContactDeliveryNotificationFailed: notifierMetrics.ContactsDeliveryNotificationsFailed.
			RegisterMeter(senderContactType, graphiteIdent, "delivery_failed"),
		ContactDeliveryNotificationCheckStopped: notifierMetrics.ContactsDeliveryNotificationsChecksStopped.
			RegisterMeter(senderContactType, graphiteIdent, "delivery_checks_stopped"),
	}
}
