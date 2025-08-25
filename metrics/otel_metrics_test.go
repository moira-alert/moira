package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOtelCounter(t *testing.T) {
	ctx := context.Background()
	exportCalled := false
	exporter := &FakeExporter{
		testExport: func(rm *metricdata.ResourceMetrics) {
			exportCalled = true
			require.NotNil(t, rm)
			require.Len(t, rm.ScopeMetrics, 1)
			scopeMetrics := rm.ScopeMetrics[0]
			require.Len(t, scopeMetrics.Metrics, 1)
			metric := scopeMetrics.Metrics[0]
			require.Equal(t, "increments", metric.Name)
			require.Equal(t, "", metric.Description)
			require.Equal(t, "", metric.Unit)

			sum, ok := metric.Data.(metricdata.Sum[int64])
			require.True(t, ok, "metric.Data should be Sum[int64]")
			require.Len(t, sum.DataPoints, 1)
			dp := sum.DataPoints[0]
			require.Equal(t, int64(10), dp.Value)
			require.Equal(t, 1, dp.Attributes.Len())
			attrSlice := dp.Attributes.ToSlice()
			for _, attr := range attrSlice {
				require.Equal(t, "custom_label", string(attr.Key))
				require.Equal(t, "test_counter", attr.Value.AsString())
			}
		},
	}

	reader := metric.NewPeriodicReader(exporter)
	metricContext := NewMetricContext(ctx)
	metricContext.AddReader(reader)

	defer func() {
		err := metricContext.Shutdown(ctx)
		require.NoError(t, err)
		require.True(t, exportCalled, "export should be called")
	}()

	registry := metricContext.CreateRegistry().WithAttributes(map[string]string {
		"custom_label": "test_counter",
	})

	counter := registry.NewCounter("increments")

	for range 10 {
		counter.Inc()
	}
}

type FakeExporter struct {
	testExport func(*metricdata.ResourceMetrics)
}

func (exp *FakeExporter) Temporality(metric.InstrumentKind) metricdata.Temporality {
	return 0
}

func (exp *FakeExporter) Aggregation(metric.InstrumentKind) metric.Aggregation {
	return metric.AggregationDefault{}
}

func (exp *FakeExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	exp.testExport(rm)
	return nil
}

func (exp *FakeExporter) ForceFlush(context.Context) error {
	return nil
}

func (exp *FakeExporter) Shutdown(context.Context) error {
	return nil
}

