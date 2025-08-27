package cmd

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelPrometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

type Telemetry struct {
	Metrics  metrics.Registry
	AttributedMetrics metrics.MetricRegistry
	stopFunc func()
}

func (source *Telemetry) Stop() {
	source.stopFunc()
}

func ConfigureTelemetry(logger moira.Logger, config TelemetryConfig, service string) (*Telemetry, error) {
	listener, err := net.Listen("tcp", config.Listen)
	serverMux := http.NewServeMux()

	if err != nil {
		return nil, err
	}

	var attributedMetrics metrics.MetricsContext = metrics.NewMetricContext(context.Background())
	var metricsRegistry metrics.Registry = metrics.NewDummyRegistry()
	if config.UseNewMetrics {
		attributedMetrics, err = configureAttributedTelemetry(logger, config, serverMux)
		if err != nil {
			return nil, err
		}
	} else {
		metricsRegistry, err = configureOldTelemetry(logger, config, service, serverMux)
		if err != nil {
			return nil, err
		}
	}

	server := &http.Server{Handler: serverMux}

	go func() {
		server.Serve(listener) //nolint
	}()

	stopServer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error().
				Error(err).
				Msg("Can't stop telemetry server correctly")
		}
	}

	return &Telemetry{
		Metrics: metricsRegistry,
		AttributedMetrics: attributedMetrics.CreateRegistry().WithAttributes(config.Otel.DefaultAttributes),
		stopFunc: func() {
			err := attributedMetrics.Shutdown(context.Background())
			stopServer()
			if err != nil {
				logger.Error().Error(err).Msg("error due to shutdown metric context")
			}
		},
	}, nil
}

func configureAttributedTelemetry(logger moira.Logger, config TelemetryConfig, serverMux *http.ServeMux) (metrics.MetricsContext, error) {

	ctx := context.Background()
	metricContext := metrics.NewMetricContext(ctx)

	if config.Pprof.Enabled {
		configurePprofServer(serverMux)
	}

	if config.Prometheus.Enabled {
		promExporter, err := otelPrometheus.New()
		if err != nil {
			return nil, err
		}
		metricContext.AddReader(promExporter)

		serverMux.Handle(config.Prometheus.MetricsPath, promhttp.Handler())
	}

	if config.Otel.Enabled {
		switch config.Otel.Protocol {
		case Grpc:
			exporter, err := otlpmetricgrpc.New(ctx,
				otlpmetricgrpc.WithEndpoint(config.Otel.CollectorURI),
				otlpmetricgrpc.WithHeaders(config.Otel.AdditionalHeaders),
			)
			if err != nil {
				return nil, err
			}
			metricContext.AddReader(metric.NewPeriodicReader(exporter))
		case Http:
			exporter, err := otlpmetrichttp.New(ctx,
				otlpmetrichttp.WithEndpoint(config.Otel.CollectorURI),
				otlpmetrichttp.WithHeaders(config.Otel.AdditionalHeaders),
			)
			if err != nil {
				return nil, err
			}
			metricContext.AddReader(metric.NewPeriodicReader(exporter))
		}
	}

	if config.Graphite.Enabled {
	}

	return metricContext, nil
}

func configureOldTelemetry(logger moira.Logger, config TelemetryConfig, service string, serverMux *http.ServeMux) (metrics.Registry, error) {
	graphiteRegistry, err := metrics.NewGraphiteRegistry(config.Graphite.GetSettings(), service)
	if err != nil {
		return nil, err
	}

	prometheusRegistry := metrics.NewPrometheusRegistry()
	prometheusRegistryAdapter := metrics.NewPrometheusRegistryAdapter(prometheusRegistry, service)

	err = configureTelemetryServer(logger, config.Listen, config.Pprof, prometheusRegistry, serverMux)
	if err != nil {
		return nil, err
	}

	return metrics.NewCompositeRegistry(graphiteRegistry, prometheusRegistryAdapter), nil
}


func configureTelemetryServer(logger moira.Logger, listen string, pprofConfig ProfilerConfig, prometheusRegistry *prometheus.Registry, serverMux *http.ServeMux) error {

	if pprofConfig.Enabled {
		configurePprofServer(serverMux)
	}

	serverMux.Handle("/metrics", promhttp.InstrumentMetricHandler(prometheusRegistry, promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{})))

	return nil
}

func configurePprofServer(serverMux *http.ServeMux) {
	serverMux.HandleFunc("/pprof/", pprof.Index)
	serverMux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	serverMux.HandleFunc("/pprof/profile", pprof.Profile)
	serverMux.HandleFunc("/pprof/symbol", pprof.Symbol)
	serverMux.HandleFunc("/pprof/trace", pprof.Trace)
	serverMux.HandleFunc("/pprof/heap", pprof.Handler("heap").ServeHTTP)
	serverMux.HandleFunc("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
}

