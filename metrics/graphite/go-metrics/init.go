package metrics

import (
	"fmt"
	"net"

	"github.com/rcrowley/go-metrics"

	goMetricsGraphite "github.com/cyberdelia/go-metrics-graphite"
	"github.com/moira-alert/moira-alert/metrics/graphite"
)

// Init is initializer for notifier graphite metrics worker based on go-metrics and go-metrics-graphite
func Init(config graphite.Config) error {
	if config.Enabled {
		address, err := net.ResolveTCPAddr("tcp", config.URI)
		if err != nil {
			return fmt.Errorf("Can not resolve graphiteURI %s: %s", config.URI, err)
		}
		go goMetricsGraphite.Graphite(metrics.DefaultRegistry, config.Interval, config.Prefix, address)
		return nil
	}
	return nil
}
