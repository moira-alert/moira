package metrics

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	goMetricsGraphite "github.com/cyberdelia/go-metrics-graphite"
	goMetrics "github.com/rcrowley/go-metrics"
)

const hostnameTmpl = "{hostname}"

type GraphiteRegistryConfig struct {
	Enabled      bool
	RuntimeStats bool
	URI          string
	Prefix       string
	Interval     time.Duration
}

type GraphiteRegistry struct {
	registry goMetrics.Registry
}

func NewDummyRegistry() Registry {
	return &GraphiteRegistry{goMetrics.NewRegistry()}
}

func NewGraphiteRegistry(config GraphiteRegistryConfig, serviceName string) (*GraphiteRegistry, error) {
	registry := goMetrics.NewRegistry()
	if config.Enabled {
		address, err := net.ResolveTCPAddr("tcp", config.URI)
		if err != nil {
			return nil, fmt.Errorf("can't resolve graphiteURI %s: %s", config.URI, err)
		}
		prefix, err := initPrefix(config.Prefix)
		if err != nil {
			return nil, fmt.Errorf("can't get OS hostname %s: %s", config.Prefix, err)
		}
		go goMetricsGraphite.Graphite(registry, config.Interval, prefix, address)
		if config.RuntimeStats {
			runtimeRegistry := goMetrics.NewPrefixedChildRegistry(registry, fmt.Sprintf("%s.", serviceName))
			goMetrics.RegisterRuntimeMemStats(runtimeRegistry)
			go goMetrics.CaptureRuntimeMemStats(runtimeRegistry, config.Interval)
		}
	}
	return &GraphiteRegistry{registry}, nil
}

func (source *GraphiteRegistry) NewTimer(name string) Timer {
	return goMetrics.NewRegisteredTimer(name, source.registry)
}

func (source *GraphiteRegistry) NewMeter(name string) Meter {
	return goMetrics.NewRegisteredMeter(name, source.registry)
}

func (source *GraphiteRegistry) NewCounter(name string) Counter {
	return goMetrics.NewRegisteredCounter(name, source.registry)
}

func (source *GraphiteRegistry) NewHistogram(name string) Histogram {
	return goMetrics.NewRegisteredHistogram(name, source.registry, goMetrics.NewExpDecaySample(1028, 0.015))
}

func (source *GraphiteRegistry) NewMetersCollection() MetersCollection {
	return &GraphiteMetersCollection{source.registry, make(map[string]goMetrics.Meter)}
}

type GraphiteMetersCollection struct {
	registry goMetrics.Registry
	metrics  map[string]goMetrics.Meter
}

func (source *GraphiteMetersCollection) RegisterMeter(name, path string) {
	source.metrics[name] = goMetrics.NewRegisteredMeter(path, source.registry)
}

func (source *GraphiteMetersCollection) GetRegisteredMeter(name string) (Meter, bool) {
	value, found := source.metrics[name]
	return value, found
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

func metricNameWithPrefix(prefix, metric string) string {
	return fmt.Sprintf("%s.%s", prefix, metric)
}
