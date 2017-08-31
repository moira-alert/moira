package metrics

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/rcrowley/go-metrics"
	"net"

	goMetricsGraphite "github.com/cyberdelia/go-metrics-graphite"
)

// Init is initializer for notifier graphite metrics worker based on go-metrics and go-metrics-graphite
func Init(config graphite.Config, logger moira.Logger, serviceName string) {
	uri := config.URI
	prefix := config.Prefix
	interval := config.Interval

	if config.Enabled {
		address, err := net.ResolveTCPAddr("tcp", uri)
		if err != nil {
			logger.Errorf("Can not resolve graphiteURI %s: %s", uri, err)
			return
		}
		go goMetricsGraphite.Graphite(metrics.DefaultRegistry, interval, fmt.Sprintf("%s.%s", prefix, serviceName), address)
	}
}
