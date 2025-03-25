package metrics

import (
	"time"
)

// NotifierMetrics is a collection of metrics used in notifier.
type NotifierMetrics struct {
	SubsMalformed                              Meter
	EventsReceived                             Meter
	EventsMalformed                            Meter
	EventsProcessingFailed                     Meter
	EventsByState                              MetersCollection
	SendingFailed                              Meter
	ContactsSendingNotificationsOK             MetersCollection
	ContactsSendingNotificationsFailed         MetersCollection
	ContactsDroppedNotifications               MetersCollection
	ContactsDeliveryNotificationsOK            MetersCollection
	ContactsDeliveryNotificationsFailed        MetersCollection
	ContactsDeliveryNotificationsChecksStopped MetersCollection
	PlotsBuildDurationMs                       Histogram
	PlotsEvaluateTriggerDurationMs             Histogram
	fetchNotificationsDurationMs               Histogram
	notifierIsAlive                            Meter
}

// ConfigureNotifierMetrics is notifier metrics configurator.
func ConfigureNotifierMetrics(registry Registry, prefix string) *NotifierMetrics {
	return &NotifierMetrics{
		SubsMalformed:                              registry.NewMeter("subs", "malformed"),
		EventsReceived:                             registry.NewMeter("events", "received"),
		EventsMalformed:                            registry.NewMeter("events", "malformed"),
		EventsProcessingFailed:                     registry.NewMeter("events", "failed"),
		EventsByState:                              NewMetersCollection(registry),
		SendingFailed:                              registry.NewMeter("sending", "failed"),
		ContactsSendingNotificationsOK:             NewMetersCollection(registry),
		ContactsSendingNotificationsFailed:         NewMetersCollection(registry),
		ContactsDroppedNotifications:               NewMetersCollection(registry),
		ContactsDeliveryNotificationsOK:            NewMetersCollection(registry),
		ContactsDeliveryNotificationsFailed:        NewMetersCollection(registry),
		ContactsDeliveryNotificationsChecksStopped: NewMetersCollection(registry),
		PlotsBuildDurationMs:                       registry.NewHistogram("plots", "build", "duration", "ms"),
		PlotsEvaluateTriggerDurationMs:             registry.NewHistogram("plots", "evaluate", "trigger", "duration", "ms"),
		fetchNotificationsDurationMs:               registry.NewHistogram("fetch", "notifications", "duration", "ms"),
		notifierIsAlive:                            registry.NewMeter("", "alive"),
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
