package metrics

// HeartBeatMetrics is a collection of metrics used in hearbeats
type HeartBeatMetrics struct {
	NotifierIsAlive Meter
}

// ConfigureHeartBeatMetrics is notifier metrics configurator
func ConfigureHeartBeatMetrics(registry Registry, serviceName string) *HeartBeatMetrics {
	return &HeartBeatMetrics{
		NotifierIsAlive: registry.NewMeter(serviceName, "alive"),
	}
}
