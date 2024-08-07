package metrics

// HeartBeatMetrics is a collection of metrics used in hearbeats.
type HeartBeatMetrics struct {
	notifierIsAlive Meter
}

// ConfigureHeartBeatMetrics is notifier metrics configurator.
func ConfigureHeartBeatMetrics(registry Registry) *HeartBeatMetrics {
	return &HeartBeatMetrics{
		notifierIsAlive: registry.NewMeter("", "alive"),
	}
}

// MarkNotifierIsAlive marks metric value.
func (hb HeartBeatMetrics) MarkNotifierIsAlive(isAlive bool) {
	if isAlive {
		hb.notifierIsAlive.Mark(1)
	}

	hb.notifierIsAlive.Mark(0)
}
