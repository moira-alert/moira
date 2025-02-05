package senders

type MetricsMarker interface {
	MarkDeliveryFailed()
}
