package metrics

import (
	"context"
	"fmt"
	"net"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type GraphiteExporter struct {
	addr string
	conn net.Conn
}

func NewGraphiteExporter(addr string) (*GraphiteExporter, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Graphite: %w", err)
	}

	return &GraphiteExporter{
		addr: addr,
		conn: conn,
	}, nil
}

func (ce *GraphiteExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	for _, scopeMetrics := range rm.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			name := sanitizeMetricName(metric.Name)

			switch data := metric.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					ce.sendLine(formatCarbonLine(withAttributes(name, dp.Attributes), float64(dp.Value), dp.Time.Unix()))
				}
			case metricdata.Sum[float64]:
				for _, dp := range data.DataPoints {
					ce.sendLine(formatCarbonLine(withAttributes(name, dp.Attributes), dp.Value, dp.Time.Unix()))
				}
			case metricdata.Gauge[int64]:
				for _, dp := range data.DataPoints {
					ce.sendLine(formatCarbonLine(withAttributes(name, dp.Attributes), float64(dp.Value), dp.Time.Unix()))
				}
			case metricdata.Gauge[float64]:
				for _, dp := range data.DataPoints {
					ce.sendLine(formatCarbonLine(withAttributes(name, dp.Attributes), dp.Value, dp.Time.Unix()))
				}
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					// count
					ce.sendLine(formatCarbonLine(withAttributes(name+".count", dp.Attributes), float64(dp.Count), dp.Time.Unix()))
					// sum
					ce.sendLine(formatCarbonLine(withAttributes(name+".sum", dp.Attributes), dp.Sum, dp.Time.Unix()))
					// buckets
					for i, bound := range dp.Bounds {
						bucketName := fmt.Sprintf("%s.bucket.le_%g", name, bound)
						ce.sendLine(formatCarbonLine(withAttributes(bucketName, dp.Attributes), float64(dp.BucketCounts[i]), dp.Time.Unix()))
					}
					// +Inf bucket
					ce.sendLine(formatCarbonLine(withAttributes(name+".bucket.le_inf", dp.Attributes), float64(dp.BucketCounts[len(dp.BucketCounts)-1]), dp.Time.Unix()))
				}
			case metricdata.ExponentialHistogram[float64]:
				for _, dp := range data.DataPoints {
					// count
					ce.sendLine(formatCarbonLine(withAttributes(name+".count", dp.Attributes), float64(dp.Count), dp.Time.Unix()))
					// sum
					ce.sendLine(formatCarbonLine(withAttributes(name+".sum", dp.Attributes), dp.Sum, dp.Time.Unix()))
					// Zero count
					ce.sendLine(formatCarbonLine(withAttributes(name+".zero_count", dp.Attributes), float64(dp.ZeroCount), dp.Time.Unix()))
					// Buckets (positive & negative)
					for offset, count := range dp.PositiveBucket.Counts {
						ce.sendLine(formatCarbonLine(withAttributes(fmt.Sprintf("%s.positive.bucket.%d", name, offset), dp.Attributes), float64(count), dp.Time.Unix()))
					}

					for offset, count := range dp.NegativeBucket.Counts {
						ce.sendLine(formatCarbonLine(withAttributes(fmt.Sprintf("%s.negative.bucket.%d", name, offset), dp.Attributes), float64(count), dp.Time.Unix()))
					}
				}
			case metricdata.Summary:
				for _, dp := range data.DataPoints {
					ce.sendLine(formatCarbonLine(withAttributes(name+".count", dp.Attributes), float64(dp.Count), dp.Time.Unix()))
					ce.sendLine(formatCarbonLine(withAttributes(name+".sum", dp.Attributes), dp.Sum, dp.Time.Unix()))

					for _, q := range dp.QuantileValues {
						qName := fmt.Sprintf("%s.quantile_%.2f", name, q.Quantile)
						ce.sendLine(formatCarbonLine(withAttributes(qName, dp.Attributes), q.Value, dp.Time.Unix()))
					}
				}
			default:
				// Unsupported
				continue
			}
		}
	}

	return nil
}

// Формат Carbon: metric.path value timestamp\n.
func formatCarbonLine(name string, value float64, timestamp int64) string {
	return fmt.Sprintf("%s %f %d\n", name, value, timestamp)
}

// Преобразует атрибуты в часть имени метрики.
func withAttributes(name string, attrs attribute.Set) string {
	if attrs.Len() == 0 {
		return name
	}

	attrParts := ""
	for _, kv := range attrs.ToSlice() {
		attrParts += fmt.Sprintf(".%s_%v", kv.Key, kv.Value.AsInterface())
	}

	return sanitizeMetricName(name + attrParts)
}

// Убирает пробелы и заменяет недопустимые символы.
func sanitizeMetricName(name string) string {
	return strings.ReplaceAll(name, " ", "_")
}

func (ge *GraphiteExporter) sendLine(line string) error {
	_, err := ge.conn.Write([]byte(line))
	return err
}

func (ge *GraphiteExporter) Shutdown(ctx context.Context) error {
	return ge.conn.Close()
}
