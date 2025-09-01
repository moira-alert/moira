package metrics

import (
	"time"
)

// NotifierMetrics is a collection of metrics used in notifier.
type NotifierMetrics struct {
	SubsMalformed                                        Meter
	EventsReceived                                       Meter
	EventsMalformed                                      Meter
	EventsProcessingFailed                               Meter
	EventsByState                                        MetersCollection
	EventsByStateAttributed                              AttributedMetricCollection
	SendingFailed                                        Meter
	ContactsSendingNotificationsOK                       MetersCollection
	ContactsSendingNotificationsOKAttributed             AttributedMetricCollection
	ContactsSendingNotificationsFailed                   MetersCollection
	ContactsSendingNotificationsFailedAttributed         AttributedMetricCollection
	ContactsDroppedNotifications                         MetersCollection
	ContactsDroppedNotificationsAttributed               AttributedMetricCollection
	ContactsDeliveryNotificationsOK                      MetersCollection
	ContactsDeliveryNotificationsOKAttributed            AttributedMetricCollection
	ContactsDeliveryNotificationsFailed                  MetersCollection
	ContactsDeliveryNotificationsFailedAttributed        AttributedMetricCollection
	ContactsDeliveryNotificationsChecksStopped           MetersCollection
	ContactsDeliveryNotificationsChecksStoppedAttributed AttributedMetricCollection
	PlotsBuildDurationMs                                 Histogram
	PlotsEvaluateTriggerDurationMs                       Histogram
	fetchNotificationsDurationMs                         Histogram
	notifierIsAlive                                      Meter
}

// ConfigureNotifierMetrics is notifier metrics configurator.
func ConfigureNotifierMetrics(registry Registry, attributedMetrics MetricRegistry, prefix string) *NotifierMetrics {
	return &NotifierMetrics{
		SubsMalformed:                                        NewCompositeMeter(registry.NewMeter("subs", "malformed"), attributedMetrics.NewGauge("subs_malformed")),
		EventsReceived:                                       NewCompositeMeter(registry.NewMeter("events", "received"), attributedMetrics.NewGauge("events_received")),
		EventsMalformed:                                      NewCompositeMeter(registry.NewMeter("events", "malformed"), attributedMetrics.NewGauge("events_malformed")),
		EventsProcessingFailed:                               NewCompositeMeter(registry.NewMeter("events", "failed"), attributedMetrics.NewGauge("events_failed")),
		EventsByState:                                        NewMetersCollection(registry),
		EventsByStateAttributed:                              NewAttributedMetricCollection(attributedMetrics),
		SendingFailed:                                        NewCompositeMeter(registry.NewMeter("sending", "failed"), attributedMetrics.NewGauge("sending_failed")),
		ContactsSendingNotificationsOK:                       NewMetersCollection(registry),
		ContactsSendingNotificationsOKAttributed:             NewAttributedMetricCollection(attributedMetrics),
		ContactsSendingNotificationsFailed:                   NewMetersCollection(registry),
		ContactsSendingNotificationsFailedAttributed:         NewAttributedMetricCollection(attributedMetrics),
		ContactsDroppedNotifications:                         NewMetersCollection(registry),
		ContactsDroppedNotificationsAttributed:               NewAttributedMetricCollection(attributedMetrics),
		ContactsDeliveryNotificationsOK:                      NewMetersCollection(registry),
		ContactsDeliveryNotificationsOKAttributed:            NewAttributedMetricCollection(attributedMetrics),
		ContactsDeliveryNotificationsFailed:                  NewMetersCollection(registry),
		ContactsDeliveryNotificationsFailedAttributed:        NewAttributedMetricCollection(attributedMetrics),
		ContactsDeliveryNotificationsChecksStopped:           NewMetersCollection(registry),
		ContactsDeliveryNotificationsChecksStoppedAttributed: NewAttributedMetricCollection(attributedMetrics),
		PlotsBuildDurationMs:                                 NewCompositeHistogram(registry.NewHistogram("plots", "build", "duration", "ms"), attributedMetrics.NewHistogram("plots_build_duration_ms")),
		PlotsEvaluateTriggerDurationMs:                       NewCompositeHistogram(registry.NewHistogram("plots", "evaluate", "trigger", "duration", "ms"), attributedMetrics.NewHistogram("plots_evaluate_trigger_duration_ms")),
		fetchNotificationsDurationMs:                         NewCompositeHistogram(registry.NewHistogram("fetch", "notifications", "duration", "ms"), attributedMetrics.NewHistogram("fetch_notifications_duration_ms")),
		notifierIsAlive:                                      NewCompositeMeter(registry.NewMeter("", "alive"), attributedMetrics.NewGauge("alive")),
	}
}

// UpdateFetchNotificationsDurationMs - counts how much time has passed since fetchNotificationsStartTime in ms and updates the metric.
func (metrics *NotifierMetrics) UpdateFetchNotificationsDurationMs(fetchNotificationsStartTime time.Time) {
	metrics.fetchNotificationsDurationMs.Update(time.Since(fetchNotificationsStartTime).Milliseconds())
}

// MarkContactDroppedNotifications marks metrics as 1 by contactType for dropped notifications.
func (metrics *NotifierMetrics) MarkContactDroppedNotifications(contactType string) {
	if metric, found := metrics.ContactsDroppedNotifications.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}
}

// MarkContactSendingNotificationOK marks metrics as 1 by contactType when notifications were successfully sent.
func (metrics *NotifierMetrics) MarkContactSendingNotificationOK(contactType string) {
	if metric, found := metrics.ContactsSendingNotificationsOK.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}
}

// MarkContactSendingNotificationFailed marks metrics as 1 by contactType when notifications were unsuccessfully sent.
func (metrics *NotifierMetrics) MarkContactSendingNotificationFailed(contactType string) {
	if metric, found := metrics.ContactsSendingNotificationsFailed.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}
}

// MarkSendingFailed marks metrics when notifications were unsuccessfully sent.
func (metrics *NotifierMetrics) MarkSendingFailed() {
	metrics.SendingFailed.Mark(1)
}

// MarkNotifierIsAlive marks metric value.
func (metrics *NotifierMetrics) MarkNotifierIsAlive(isAlive bool) {
	if isAlive {
		metrics.notifierIsAlive.Mark(1)
		return
	}

	metrics.notifierIsAlive.Mark(0)
}
