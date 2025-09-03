package metrics

// SenderMetrics should be used for sender which can understand if the notification was delivered or not.
type SenderMetrics struct {
	ContactDeliveryNotificationOK           Meter
	ContactDeliveryNotificationFailed       Meter
	ContactDeliveryNotificationCheckStopped Meter
}

// ConfigureSenderMetrics configures SenderMetrics using NotifierMetrics with given graphiteIdent for senderContactType.
func ConfigureSenderMetrics(notifierMetrics *NotifierMetrics, graphiteIdent string, senderContactType string) (*SenderMetrics, error) {
	const senderContactTypeField = "sender_contact_type"

	deliveryOk, err := notifierMetrics.ContactsDeliveryNotificationsOKAttributed.RegisterMeter("delivery_ok", Attributes{
		Attribute{Key: senderContactTypeField, Value: graphiteIdent},
	})
	if err != nil {
		return nil, err
	}

	deliveryFailed, err := notifierMetrics.ContactsDeliveryNotificationsFailedAttributed.RegisterMeter("delivery_failed", Attributes{
		Attribute{Key: senderContactTypeField, Value: graphiteIdent},
	})
	if err != nil {
		return nil, err
	}

	deliveryChecksStopped, err := notifierMetrics.ContactsDeliveryNotificationsChecksStoppedAttributed.RegisterMeter("delivery_checks_stopped", Attributes{
		Attribute{Key: senderContactTypeField, Value: graphiteIdent},
	})
	if err != nil {
		return nil, err
	}

	return &SenderMetrics{
		ContactDeliveryNotificationOK: NewCompositeMeter(
			notifierMetrics.ContactsDeliveryNotificationsOK.RegisterMeter(senderContactType, graphiteIdent, "delivery_ok"),
			deliveryOk,
		),
		ContactDeliveryNotificationFailed: NewCompositeMeter(
			notifierMetrics.ContactsDeliveryNotificationsFailed.RegisterMeter(senderContactType, graphiteIdent, "delivery_failed"),
			deliveryFailed,
		),
		ContactDeliveryNotificationCheckStopped: NewCompositeMeter(
			notifierMetrics.ContactsDeliveryNotificationsChecksStopped.RegisterMeter(senderContactType, graphiteIdent, "delivery_checks_stopped"),
			deliveryChecksStopped,
		),
	}, nil
}
