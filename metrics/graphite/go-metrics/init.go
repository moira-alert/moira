package metrics

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/rcrowley/go-metrics"
	"net"
	"os"
	"strings"

	goMetricsGraphite "github.com/cyberdelia/go-metrics-graphite"
)

//Init is initializer for notifier graphite metrics worker based on go-metrics and go-metrics-graphite
func Init(config graphite.Config, logger moira.Logger, serviceName string) {
	uri := config.URI
	prefix := config.Prefix
	interval := config.Interval

	if uri != "" {
		address, err := net.ResolveTCPAddr("tcp", uri)
		if err != nil {
			logger.Errorf("Can not resolve graphiteURI %s: %s", uri, err)
			return
		}
		hostname, err := os.Hostname()
		if err != nil {
			logger.Errorf("Can not get OS hostname: %s", err)
			return
		}
		shortName := strings.Split(hostname, ".")[0]
		go goMetricsGraphite.Graphite(metrics.DefaultRegistry, interval, fmt.Sprintf("%s.%s.%s", prefix, serviceName, shortName), address)
	}
}
