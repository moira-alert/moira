package metrics

import (
	"fmt"
	goMetrics "github.com/rcrowley/go-metrics"
)

func newRuntimeMetricsRegistry(prefix string, runtimeMetricsEnabled bool) *goMetrics.Registry {
	if runtimeMetricsEnabled {
		registryPrefix := fmt.Sprintf("%s.", prefix)
		registry := goMetrics.NewPrefixedRegistry(registryPrefix)
		return &registry
	}
	return nil
}
