package cmd

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"sync"
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
		defaultAttributes = append(defaultAttributes, metrics.Attribute{Key: k, Value: v})
	}

	attributedRegistry := attributedMetrics.CreateRegistry().WithAttributes(defaultAttributes)

	if config.RuntimeStats {
		exit := make(chan struct{})
		waitGr := sync.WaitGroup{}
		waitGr.Add(1)

		err := configureRuntimeStats(attributedRegistry, exit, &waitGr)
		if err != nil {
			return nil, err
		}

		stopServer = func() {
			stopServer()
			exit <- struct{}{}

			waitGr.Wait()
		}
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

	return metrics.NewCompositeRegistry(metricRegistries...), metricContext, nil
}

func configureRuntimeStats(registry metrics.MetricRegistry, exit <-chan struct{}, waitGr *sync.WaitGroup) error { //nolint:gocyclo
	alloc, err := registry.NewGauge("runtime_alloc")
	if err != nil {
		return err
	}

	buckHashSys, err := registry.NewGauge("runtime_buckHashSys")
	if err != nil {
		return err
	}

	debugGC, err := registry.NewGauge("runtime_debugGC")
	if err != nil {
		return err
	}

	enableGC, err := registry.NewGauge("runtime_enableGC")
	if err != nil {
		return err
	}

	frees, err := registry.NewGauge("runtime_frees")
	if err != nil {
		return err
	}

	heapAlloc, err := registry.NewGauge("runtime_heapAlloc")
	if err != nil {
		return err
	}

	numGoroutines, err := registry.NewGauge("runtime_goroutines")
	if err != nil {
		return err
	}

	numCgoCall, err := registry.NewGauge("runtime_numCgoCall")
	if err != nil {
		return err
	}

	heapSys, err := registry.NewGauge("runtime_heapSys")
	if err != nil {
		return err
	}

	heapIdle, err := registry.NewGauge("runtime_heapIdle")
	if err != nil {
		return err
	}

	heapInuse, err := registry.NewGauge("runtime_heapInuse")
	if err != nil {
		return err
	}

	heapReleased, err := registry.NewGauge("runtime_heapReleased")
	if err != nil {
		return err
	}

	heapObjects, err := registry.NewGauge("runtime_heapObjects")
	if err != nil {
		return err
	}

	stackInuse, err := registry.NewGauge("runtime_stackInuse")
	if err != nil {
		return err
	}

	stackSys, err := registry.NewGauge("runtime_stackSys")
	if err != nil {
		return err
	}

	mSpanInuse, err := registry.NewGauge("runtime_mSpanInuse")
	if err != nil {
		return err
	}

	mSpanSys, err := registry.NewGauge("runtime_mSpanSys")
	if err != nil {
		return err
	}

	mCacheInuse, err := registry.NewGauge("runtime_mCacheInuse")
	if err != nil {
		return err
	}

	mCacheSys, err := registry.NewGauge("runtime_mCacheSys")
	if err != nil {
		return err
	}

	buckHashSysGauge, err := registry.NewGauge("runtime_buckHashSysGauge")
	if err != nil {
		return err
	}

	gcSys, err := registry.NewGauge("runtime_gcSys")
	if err != nil {
		return err
	}

	otherSys, err := registry.NewGauge("runtime_otherSys")
	if err != nil {
		return err
	}

	nextGC, err := registry.NewGauge("runtime_nextGC")
	if err != nil {
		return err
	}

	lastGC, err := registry.NewGauge("runtime_lastGC")
	if err != nil {
		return err
	}

	pauseTotalNs, err := registry.NewGauge("runtime_pauseTotalNs")
	if err != nil {
		return err
	}

	numGC, err := registry.NewGauge("runtime_numGC")
	if err != nil {
		return err
	}

	lookups, err := registry.NewGauge("runtime_lookups")
	if err != nil {
		return err
	}

	mallocs, err := registry.NewGauge("runtime_mallocs")
	if err != nil {
		return err
	}

	totalAllocs, err := registry.NewGauge("runtime_totalAllocs")
	if err != nil {
		return err
	}

	go func() {
	LOOP:
		for {
			select {
			case <-time.Tick(time.Minute):
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)

				alloc.Mark(int64(mem.Alloc))
				buckHashSys.Mark(int64(mem.BuckHashSys))
				if mem.DebugGC {
					debugGC.Mark(1)
				} else {
					debugGC.Mark(0)
				}
				if mem.EnableGC {
					enableGC.Mark(1)
				} else {
					enableGC.Mark(0)
				}
				frees.Mark(int64(mem.Frees))
				heapAlloc.Mark(int64(mem.HeapAlloc))
				numGoroutines.Mark(int64(runtime.NumGoroutine()))
				numCgoCall.Mark(runtime.NumCgoCall())
				heapSys.Mark(int64(mem.HeapSys))
				heapIdle.Mark(int64(mem.HeapIdle))
				heapInuse.Mark(int64(mem.HeapInuse))
				heapReleased.Mark(int64(mem.HeapReleased))
				heapObjects.Mark(int64(mem.HeapObjects))
				stackInuse.Mark(int64(mem.StackInuse))
				stackSys.Mark(int64(mem.StackSys))
				mSpanInuse.Mark(int64(mem.MSpanInuse))
				mSpanSys.Mark(int64(mem.MSpanSys))
				mCacheInuse.Mark(int64(mem.MCacheInuse))
				mCacheSys.Mark(int64(mem.MCacheSys))
				buckHashSysGauge.Mark(int64(mem.BuckHashSys))
				gcSys.Mark(int64(mem.GCSys))
				otherSys.Mark(int64(mem.OtherSys))
				nextGC.Mark(int64(mem.NextGC))
				lastGC.Mark(int64(mem.LastGC))
				pauseTotalNs.Mark(int64(mem.PauseTotalNs))
				numGC.Mark(int64(mem.NumGC))
				lookups.Mark(int64(mem.Lookups))
				mallocs.Mark(int64(mem.Mallocs))
				totalAllocs.Mark(int64(mem.TotalAlloc))
			case <-exit:
				waitGr.Done()
				break LOOP
			}
		}
	}()

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
