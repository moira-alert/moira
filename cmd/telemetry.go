package cmd

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

type Telemetry struct {
	Metrics  metrics.Registry
	stopFunc func()
}

func (source *Telemetry) Stop() {
	source.stopFunc()
}

func ConfigureTelemetry(logger moira.Logger, config TelemetryConfig, service string) (*Telemetry, error) {
	graphiteRegistry, err := metrics.NewGraphiteRegistry(config.Graphite.GetSettings(), service)
	if err != nil {
		return nil, err
	}
	prometheusRegistry := metrics.NewPrometheusRegistry()
	prometheusRegistryAdapter := metrics.NewPrometheusRegistryAdapter(prometheusRegistry, service)
	stopServer, err := startTelemetryServer(logger, config.Listen, config.Pprof, prometheusRegistry)
	if err != nil {
		return nil, err
	}
	return &Telemetry{Metrics: metrics.NewCompositeRegistry(graphiteRegistry, prometheusRegistryAdapter), stopFunc: stopServer}, nil
}

func startTelemetryServer(logger moira.Logger, listen string, pprofConfig ProfilerConfig, prometheusRegistry *prometheus.Registry) (func(), error) {
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}
	serverMux := http.NewServeMux()
	if pprofConfig.Enabled {
		serverMux.HandleFunc("/pprof/", pprof.Index)
		serverMux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
		serverMux.HandleFunc("/pprof/profile", pprof.Profile)
		serverMux.HandleFunc("/pprof/symbol", pprof.Symbol)
		serverMux.HandleFunc("/pprof/trace", pprof.Trace)
		serverMux.HandleFunc("/pprof/heap", pprof.Handler("heap").ServeHTTP)
		serverMux.HandleFunc("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	}
	serverMux.Handle("/metrics", promhttp.InstrumentMetricHandler(prometheusRegistry, promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{})))
	server := &http.Server{Handler: serverMux}
	go func() {
		server.Serve(listener)
	}()
	stopServer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Errorf("Can't stop telemetry server correctly: %v", err)
		}
	}
	return stopServer, nil
}
