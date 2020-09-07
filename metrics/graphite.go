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
		if config.RuntimeStats {
			goMetrics.RegisterRuntimeMemStats(registry)
			go goMetrics.CaptureRuntimeMemStats(registry, config.Interval)
		}
		go goMetricsGraphite.Graphite(registry, config.Interval, getGraphiteMetricName([]string{prefix, serviceName}), address)
	}
	return &GraphiteRegistry{registry}, nil
}

func (source *GraphiteRegistry) NewTimer(path ...string) Timer {
	return goMetrics.NewRegisteredTimer(getGraphiteMetricName(path), source.registry)
}

func (source *GraphiteRegistry) NewMeter(path ...string) Meter {
	return goMetrics.NewRegisteredMeter(getGraphiteMetricName(path), source.registry)
}

func (source *GraphiteRegistry) NewCounter(path ...string) Counter {
	return &graphiteCounter{goMetrics.NewRegisteredCounter(getGraphiteMetricName(path), source.registry)}
}

func (source *GraphiteRegistry) NewHistogram(path ...string) Histogram {
	return goMetrics.NewRegisteredHistogram(getGraphiteMetricName(path), source.registry, goMetrics.NewExpDecaySample(1028, 0.015)) //nolint
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

type graphiteCounter struct {
	counter goMetrics.Counter
}

func (source *graphiteCounter) Inc() {
	source.counter.Inc(1)
}

func (source *graphiteCounter) Count() int64 {
	return source.counter.Count()
}

func getGraphiteMetricName(path []string) string {
	return strings.Join(path, ".")
}
