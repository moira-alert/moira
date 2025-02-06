package metrics

// SenderMetrics should be used for sender which can understand if the notification was delivered or not.
type SenderMetrics struct {
	SenderDeliveryOK     Meter
	SenderDeliveryFailed Meter
}

// ConfigureSenderMetrics configures SenderMetrics using NotifierMetrics with given graphiteIdent for senderContactType.
func ConfigureSenderMetrics(notifierMetrics *NotifierMetrics, graphiteIdent string, senderContactType string) *SenderMetrics {
	return &SenderMetrics{
		SenderDeliveryOK:     notifierMetrics.SendersDeliveryOK.RegisterMeter(senderContactType, graphiteIdent, "delivery_ok"),
		SenderDeliveryFailed: notifierMetrics.SendersDeliveryFailed.RegisterMeter(senderContactType, graphiteIdent, "delivery_failed"),
	}
}
