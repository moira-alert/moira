package cmd

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelPrometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

type Telemetry struct {
	Metrics           metrics.Registry
	AttributedMetrics metrics.MetricRegistry
	stopFunc          func()
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

	metricsRegistry, attributedMetrics, err := configureTelemetry(config, service, serverMux)
	if err != nil {
		return nil, err
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

	defaultAttributes := metrics.Attributes{}

	for k, v := range config.DefaultAttributes {
		k = replaceStaticTemplate(k)
		v = replaceStaticTemplate(v)
		defaultAttributes = append(defaultAttributes, metrics.Attribute{Key: k, Value: v})
	}

	attributedRegistry, err := attributedMetrics.CreateRegistry(defaultAttributes...)
	if err != nil {
		return nil, err
	}

	return &Telemetry{
		Metrics:           metricsRegistry,
		AttributedMetrics: attributedRegistry,
		stopFunc: func() {
			err := attributedMetrics.Shutdown(context.Background())

			stopServer()

			if err != nil {
				logger.Error().Error(err).Msg("error due to shutdown metric context")
			}
		},
	}, nil
}

func configureTelemetry(config TelemetryConfig, service string, serverMux *http.ServeMux) (metrics.Registry, metrics.MetricsContext, error) {
	ctx := context.Background()
	metricContext := metrics.NewMetricContext(ctx)
	metricRegistries := []metrics.Registry{}

	if config.Pprof.Enabled {
		configurePprofServer(serverMux)
	}

	if config.Prometheus.Enabled {
		prometheusRegistry := metrics.NewPrometheusRegistry()

		if config.UseNewMetrics {
			promExporter, err := otelPrometheus.New(otelPrometheus.WithRegisterer(prometheusRegistry))
			if err != nil {
				return nil, nil, err
			}

			metricContext.AddReader(promExporter)
		}

		prometheusRegistryAdapter := metrics.NewPrometheusRegistryAdapter(prometheusRegistry, service)
		metricRegistries = append(metricRegistries, prometheusRegistryAdapter)

		serverMux.Handle(config.Prometheus.MetricsPath, promhttp.InstrumentMetricHandler(prometheusRegistry, promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{})))
	}

	if config.Graphite.Enabled {
		graphiteRegistry, err := metrics.NewGraphiteRegistry(config.Graphite.GetSettings(), service)
		if err != nil {
			return nil, nil, err
		}

		metricRegistries = append(metricRegistries, graphiteRegistry)
	}

	if config.Otel.Enabled {
		switch config.Otel.Protocol {
		case Grpc:
			options := []otlpmetricgrpc.Option{
				otlpmetricgrpc.WithEndpoint(config.Otel.CollectorURI),
				otlpmetricgrpc.WithHeaders(config.Otel.AdditionalHeaders),
			}
			if config.Otel.Insecure {
				options = append(options, otlpmetricgrpc.WithInsecure())
			}

			exporter, err := otlpmetricgrpc.New(ctx, options...)
			if err != nil {
				return nil, nil, err
			}

			metricContext.AddReader(metric.NewPeriodicReader(exporter, metric.WithInterval(config.Otel.PushInterval)))
		case Http:
			exporter, err := otlpmetrichttp.New(ctx,
				otlpmetrichttp.WithEndpoint(config.Otel.CollectorURI),
				otlpmetrichttp.WithHeaders(config.Otel.AdditionalHeaders),
			)
			if err != nil {
				return nil, nil, err
			}

			metricContext.AddReader(metric.NewPeriodicReader(exporter, metric.WithInterval(config.Otel.PushInterval)))
		}
	}

	if config.RuntimeStats {
		metricContext.AddRuntimeStats(time.Second)
	}

	return metrics.NewCompositeRegistry(metricRegistries...), metricContext, nil
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

const (
	hostnameTemplate string = "{hostname}"
)

func hostnameTemplateReplacer() string {
	name, err := os.Hostname()
	if err != nil {
		return hostnameTemplate
	}

	return name
}

var templateItems map[string]string = map[string]string{
	hostnameTemplate: hostnameTemplateReplacer(),
}

func replaceStaticTemplate(input string) string {
	res := input
	for template, replacement := range templateItems {
		res = strings.ReplaceAll(res, template, replacement)
	}

	return res
}
