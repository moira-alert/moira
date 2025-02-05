package metrics

import (
	"time"
)

// NotifierMetrics is a collection of metrics used in notifier.
type NotifierMetrics struct {
	SubsMalformed                  Meter
	EventsReceived                 Meter
	EventsMalformed                Meter
	EventsProcessingFailed         Meter
	EventsByState                  MetersCollection
	SendingFailed                  Meter
	SendersOkMetrics               MetersCollection
	SendersFailedMetrics           MetersCollection
	SendersDroppedNotifications    MetersCollection
	SendersDeliveryFailed          MetersCollection
	PlotsBuildDurationMs           Histogram
	PlotsEvaluateTriggerDurationMs Histogram
	fetchNotificationsDurationMs   Histogram
	notifierIsAlive                Meter
}

// ConfigureNotifierMetrics is notifier metrics configurator.
func ConfigureNotifierMetrics(registry Registry, prefix string) *NotifierMetrics {
	return &NotifierMetrics{
		SubsMalformed:                  registry.NewMeter("subs", "malformed"),
		EventsReceived:                 registry.NewMeter("events", "received"),
		EventsMalformed:                registry.NewMeter("events", "malformed"),
		EventsProcessingFailed:         registry.NewMeter("events", "failed"),
		EventsByState:                  NewMetersCollection(registry),
		SendingFailed:                  registry.NewMeter("sending", "failed"),
		SendersOkMetrics:               NewMetersCollection(registry),
		SendersFailedMetrics:           NewMetersCollection(registry),
		SendersDroppedNotifications:    NewMetersCollection(registry),
		SendersDeliveryFailed:          NewMetersCollection(registry),
		PlotsBuildDurationMs:           registry.NewHistogram("plots", "build", "duration", "ms"),
		PlotsEvaluateTriggerDurationMs: registry.NewHistogram("plots", "evaluate", "trigger", "duration", "ms"),
		fetchNotificationsDurationMs:   registry.NewHistogram("fetch", "notifications", "duration", "ms"),
		notifierIsAlive:                registry.NewMeter("", "alive"),
	}
}

// UpdateFetchNotificationsDurationMs - counts how much time has passed since fetchNotificationsStartTime in ms and updates the metric.
func (metrics *NotifierMetrics) UpdateFetchNotificationsDurationMs(fetchNotificationsStartTime time.Time) {
	metrics.fetchNotificationsDurationMs.Update(time.Since(fetchNotificationsStartTime).Milliseconds())
}

// MarkSendersDroppedNotifications marks metrics as 1 by contactType for dropped notifications.
func (metrics *NotifierMetrics) MarkSendersDroppedNotifications(contactType string) {
	if metric, found := metrics.SendersDroppedNotifications.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}
}

// MarkSendersOkMetrics marks metrics as 1 by contactType when notifications were successfully sent.
func (metrics *NotifierMetrics) MarkSendersOkMetrics(contactType string) {
	if metric, found := metrics.SendersOkMetrics.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}
}

// MarkSendersFailedMetrics marks metrics as 1 by contactType when notifications were unsuccessfully sent.
func (metrics *NotifierMetrics) MarkSendersFailedMetrics(contactType string) {
	if metric, found := metrics.SendersFailedMetrics.GetRegisteredMeter(contactType); found {
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

// MarkDeliveryFailed marks metric as 1 by contact type for not delivered notifications.
func (metrics *NotifierMetrics) MarkDeliveryFailed(contactType string) {
	if metric, found := metrics.SendersDeliveryFailed.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}
}
