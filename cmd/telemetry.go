package cmd

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

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

func ConfigureTelemetry(logger moira.Logger, config TelemetryConfig, serviceName string) (*Telemetry, error) {
	graphiteMetricsRegistry, err := metrics.NewGraphiteRegistry(config.Graphite.GetSettings(), serviceName)
	if err != nil {
		return nil, err
	}

	stopServer, err := startTelemetryServer(logger, config.Listen, config.Pprof)
	if err != nil {
		return nil, err
	}
	return &Telemetry{Metrics: graphiteMetricsRegistry, stopFunc: stopServer}, nil
}

func startTelemetryServer(logger moira.Logger, listen string, pprofConfig ProfilerConfig) (func(), error) {

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
	}
	server := &http.Server{Handler: serverMux}
	go func() {
		server.Serve(listener)
	}()
	stopServer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Errorf("Can't stop telemetry server correctly: %v", err)
		}
	}
	return stopServer, nil
}
