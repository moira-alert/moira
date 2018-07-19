package metrics

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/rcrowley/go-metrics"

	goMetricsGraphite "github.com/cyberdelia/go-metrics-graphite"
	"github.com/moira-alert/moira/metrics/graphite"
)

const hostnameTmpl = "{hostname}"

// Init is initializer for notifier graphite metrics worker based on go-metrics and go-metrics-graphite
func Init(config graphite.Config, serviceName string) error {
	if config.Enabled {
		address, err := net.ResolveTCPAddr("tcp", config.URI)
		if err != nil {
			return fmt.Errorf("can't resolve graphiteURI %s: %s", config.URI, err)
		}
		prefix, err := initPrefix(config.Prefix)
		if err != nil {
			return fmt.Errorf("can't get OS hostname %s: %s", config.Prefix, err)
		}
		go goMetricsGraphite.Graphite(metrics.DefaultRegistry, config.Interval, prefix, address)
		if config.RuntimeStats {
			runtimeRegistry := initRuntimeRegistry(serviceName)
			go metrics.CaptureRuntimeMemStats(runtimeRegistry, config.Interval)
			go goMetricsGraphite.Graphite(runtimeRegistry, config.Interval, prefix, address)
		}
		return nil
	}
	return nil
}

func initRuntimeRegistry(serviceName string) metrics.Registry {
	runtimePrefix := fmt.Sprintf("%s.", serviceName)
	runtimeRegistry := metrics.NewPrefixedRegistry(runtimePrefix)
	metrics.RegisterRuntimeMemStats(runtimeRegistry)
	return runtimeRegistry
}

func initPrefix(prefix string) (string, error) {
	if !strings.Contains(prefix, hostnameTmpl) {
		return prefix, nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		return prefix, err
	}
	short := strings.Split(hostname, ".")[0]
	return strings.Replace(prefix, hostnameTmpl, short, -1), nil
}
