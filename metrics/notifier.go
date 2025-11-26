package metrics

import (
	"time"
)

// NotifierMetrics is a collection of metrics used in notifier.
type NotifierMetrics struct {
	// Depricated: use counter instead
	SubsMalformed                                        Meter
	SubsMalformedCounter Counter
	// Depricated: use counter instead
	EventsReceived                                       Meter
	EventsReceivedCounter Counter
	// Depricated: use counter instead
	EventsMalformed                                      Meter
	EventsMalformedCounter Counter
	// Depricated: use counter instead
	EventsProcessingFailed                               Meter
	EventsProcessingFailedCounter Counter
	EventsByState                                        MetersCollection
	EventsByStateAttributed                              AttributedMetricCollection
	// Depricated: use counter instead
	SendingFailed                                        Meter
	SendingFailedCounter Counter
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
func ConfigureNotifierMetrics(registry Registry, attributedRegistry MetricRegistry, prefix string) (*NotifierMetrics, error) {
	subsMalformed, err := attributedRegistry.NewCounter("subs.malformed")
	if err != nil {
		return nil, err
	}

	eventsMalformed, err := attributedRegistry.NewCounter("events.malformed")
	if err != nil {
		return nil, err
	}

	eventsReceived, err := attributedRegistry.NewCounter("events.received")
	if err != nil {
		return nil, err
	}

	eventsProcessingFailed, err := attributedRegistry.NewCounter("events.failed_processing")
	if err != nil {
		return nil, err
	}

	sendingFailed, err := attributedRegistry.NewCounter("notifications.sending.failed")
	if err != nil {
		return nil, err
	}

	plotsBuildDurationMs, err := attributedRegistry.NewHistogram("plots.build.duration_ms")
	if err != nil {
		return nil, err
	}

	plotsEvaluateTriggerDurationMs, err := attributedRegistry.NewHistogram("plots.evaluate_trigger.duration_ms")
	if err != nil {
		return nil, err
	}

	fetchNotificationsDurationMs, err := attributedRegistry.NewHistogram("notifications.fetch.duration_ms")
	if err != nil {
		return nil, err
	}

	notifierIsAlive, err := attributedRegistry.NewGauge("alive")
	if err != nil {
		return nil, err
	}

	return &NotifierMetrics{
		SubsMalformed:                                        registry.NewMeter("subs", "malformed"),
		SubsMalformedCounter: subsMalformed,
		EventsReceived:                                       registry.NewMeter("events", "received"),
		EventsReceivedCounter: eventsReceived,
		EventsMalformed:                                      registry.NewMeter("events", "malformed"),
		EventsMalformedCounter: eventsMalformed,
		EventsProcessingFailed:                               registry.NewMeter("events", "failed"),
		EventsProcessingFailedCounter: eventsProcessingFailed,
		EventsByState:                                        NewMetersCollection(registry),
		EventsByStateAttributed:                              NewAttributedMetricCollection(attributedRegistry),
		SendingFailed:                                        registry.NewMeter("sending", "failed"),
		SendingFailedCounter: sendingFailed,
		ContactsSendingNotificationsOK:                       NewMetersCollection(registry),
		ContactsSendingNotificationsOKAttributed:             NewAttributedMetricCollection(attributedRegistry),
		ContactsSendingNotificationsFailed:                   NewMetersCollection(registry),
		ContactsSendingNotificationsFailedAttributed:         NewAttributedMetricCollection(attributedRegistry),
		ContactsDroppedNotifications:                         NewMetersCollection(registry),
		ContactsDroppedNotificationsAttributed:               NewAttributedMetricCollection(attributedRegistry),
		ContactsDeliveryNotificationsOK:                      NewMetersCollection(registry),
		ContactsDeliveryNotificationsOKAttributed:            NewAttributedMetricCollection(attributedRegistry),
		ContactsDeliveryNotificationsFailed:                  NewMetersCollection(registry),
		ContactsDeliveryNotificationsFailedAttributed:        NewAttributedMetricCollection(attributedRegistry),
		ContactsDeliveryNotificationsChecksStopped:           NewMetersCollection(registry),
		ContactsDeliveryNotificationsChecksStoppedAttributed: NewAttributedMetricCollection(attributedRegistry),
		PlotsBuildDurationMs:                                 NewCompositeHistogram(registry.NewHistogram("plots", "build", "duration", "ms"), plotsBuildDurationMs),
		PlotsEvaluateTriggerDurationMs:                       NewCompositeHistogram(registry.NewHistogram("plots", "evaluate", "trigger", "duration", "ms"), plotsEvaluateTriggerDurationMs),
		fetchNotificationsDurationMs:                         NewCompositeHistogram(registry.NewHistogram("fetch", "notifications", "duration", "ms"), fetchNotificationsDurationMs),
		notifierIsAlive:                                      NewCompositeMeter(registry.NewMeter("", "alive"), notifierIsAlive),
	}, nil
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

	if metric, found := metrics.ContactsDroppedNotificationsAttributed.GetRegisteredCounter(contactType); found {
		metric.Inc()
	}
}

// MarkContactSendingNotificationOK marks metrics as 1 by contactType when notifications were successfully sent.
func (metrics *NotifierMetrics) MarkContactSendingNotificationOK(contactType string) {
	if metric, found := metrics.ContactsSendingNotificationsOK.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}

	if metric, found := metrics.ContactsSendingNotificationsOKAttributed.GetRegisteredCounter(contactType); found {
		metric.Inc()
	}
}

// MarkContactSendingNotificationFailed marks metrics as 1 by contactType when notifications were unsuccessfully sent.
func (metrics *NotifierMetrics) MarkContactSendingNotificationFailed(contactType string) {
	if metric, found := metrics.ContactsSendingNotificationsFailed.GetRegisteredMeter(contactType); found {
		metric.Mark(1)
	}

	if metric, found := metrics.ContactsSendingNotificationsFailedAttributed.GetRegisteredCounter(contactType); found {
		metric.Inc()
	}
}

// MarkSendingFailed marks metrics when notifications were unsuccessfully sent.
func (metrics *NotifierMetrics) MarkSendingFailed() {
	metrics.SendingFailed.Mark(1)
	metrics.SendingFailedCounter.Inc()
}

// MarkNotifierIsAlive marks metric value.
func (metrics *NotifierMetrics) MarkNotifierIsAlive(isAlive bool) {
	if isAlive {
		metrics.notifierIsAlive.Mark(1)
		return
	}

	metrics.notifierIsAlive.Mark(0)
}
