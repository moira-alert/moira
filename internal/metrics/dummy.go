package metrics

import goMetrics "github.com/rcrowley/go-metrics"

func NewDummyRegistry() Registry {
	return &GraphiteRegistry{goMetrics.NewRegistry()}
}
