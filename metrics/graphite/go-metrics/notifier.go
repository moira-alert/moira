package go_metrics

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/rcrowley/go-metrics"
	"net"
	"os"
	"strings"
	"time"

	goMetricsGraphite "github.com/cyberdelia/go-metrics-graphite"
)

var NotifierMetric NotifierMetrics

type MetricsHash map[string]graphite.Meter

type NotifierMetrics struct {
	config                 graphite.Config
	registry               metrics.Registry
	EventsReceived         graphite.Meter
	EventsMalformed        graphite.Meter
	EventsProcessingFailed graphite.Meter
	SubsMalformed          graphite.Meter
	SendingFailed          graphite.Meter
	SendersOkMetrics       MetricsHash
	SendersFailedMetrics   MetricsHash
}

func ConfigureNotifierMetrics(config graphite.Config) NotifierMetrics {
	registry := metrics.NewRegistry()
	return NotifierMetrics{
		config:                 config,
		registry:               registry,
		EventsReceived:         metrics.NewRegisteredMeter("events.received", registry),
		EventsMalformed:        metrics.NewRegisteredMeter("events.malformed", registry),
		EventsProcessingFailed: metrics.NewRegisteredMeter("events.failed", registry),
		SubsMalformed:          metrics.NewRegisteredMeter("subs.malformed", registry),
		SendingFailed:          metrics.NewRegisteredMeter("sending.failed", registry),
		SendersOkMetrics:       make(map[string]graphite.Meter),
		SendersFailedMetrics:   make(map[string]graphite.Meter),
	}
}

func (metric *NotifierMetrics) Init(logger moira_alert.Logger) {
	uri := metric.config.URI
	prefix := metric.config.Prefix
	interval := metric.config.Interval

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
		go goMetricsGraphite.Graphite(metric.registry, time.Duration(interval)*time.Second, fmt.Sprintf("%s.notifier.%s", prefix, shortName), address)
	}
}
