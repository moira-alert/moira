package local

import (
	"fmt"
	"math"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func BenchmarkUnpackMetricsValues(b *testing.B) {
	var from int64 = 17
	var until int64 = 1317
	var retention int64 = 10

	timer := NewTimerRoundingTimestamps(from, until, retention)

	metricsCount := 7300

	metricsValues := make([]*moira.MetricValue, 0)

	for i := from + retention; i <= until; i += retention {
		metricsValues = append(metricsValues, &moira.MetricValue{
			RetentionTimestamp: (i / retention) * retention,
			Timestamp:          i,
			Value:              float64(i),
		})
	}
	metricData := map[string][]*moira.MetricValue{"metric1": metricsValues}
	for i := 0; i < metricsCount; i++ {
		metricData[fmt.Sprintf("metric%v", i)] = metricsValues
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unpackMetricsValues(metricData, timer)
	}
}

func BenchmarkUnpackMetricValues(b *testing.B) {
	var from int64 = 17
	var until int64 = 317
	var retention int64 = 10

	timer := NewTimerRoundingTimestamps(from, until, retention)

	metricsValues := make([]*moira.MetricValue, 0)

	for i := from + retention; i <= until; i += retention {
		metricsValues = append(metricsValues, &moira.MetricValue{
			RetentionTimestamp: (i / retention) * retention,
			Timestamp:          i,
			Value:              float64(i),
		})
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unpackMetricValues(metricsValues, timer)
	}
}

func TestFetchDataErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	fetchData := fetchData{database: database}

	pattern := "super-puper-pattern"
	metric := "super-puper-metric"

	timer := NewTimerRoundingTimestamps(17, 67, 10)

	retentionErr := fmt.Errorf("Ooops, retention error")
	patternErr := fmt.Errorf("Ooops, pattern error")
	metricErr := fmt.Errorf("Ooops, metric error")

	Convey("GetPatternMetricsError", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return(nil, patternErr)

		metrics, err := fetchData.fetchMetricNames(pattern)
		So(metrics, ShouldBeNil)
		So(err, ShouldResemble, patternErr)
	})

	Convey("GetMetricRetentionError", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(int64(0), retentionErr)

		metrics, err := fetchData.fetchMetricNames(pattern)
		So(metrics, ShouldBeNil)
		So(err, ShouldResemble, retentionErr)
	})

	Convey("GetMetricsValuesError", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(timer.retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, timer.from, timer.until-1).Return(nil, metricErr)

		metrics, err := fetchData.fetchMetricNames(pattern)

		expectedMetrics := metricsWithRetention{
			retention: timer.retention,
			metrics:   []string{metric},
		}

		So(*metrics, ShouldResemble, expectedMetrics)
		So(err, ShouldBeNil)

		metricData, err := fetchData.fetchMetricValues(pattern, metrics, timer)

		So(metricData, ShouldBeNil)
		So(err, ShouldResemble, metricErr)
	})
}

func TestFetchData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	fetchData := fetchData{database: dataBase}

	pattern := "super-puper-pattern"
	metric := "super-puper-metric"

	dataList := map[string][]*moira.MetricValue{
		metric: generateMetricValues(20, 23, 10, 5),
	}

	var from int64 = 17
	var until int64 = 57
	var retention int64 = 10
	timer := NewTimerRoundingTimestamps(from, until, retention)

	Convey("Test no metrics", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)

		metrics, err := fetchData.fetchMetricNames(pattern)
		So(err, ShouldBeNil)

		timer := NewTimerRoundingTimestamps(from, until, metrics.retention)
		metricValues, err := fetchData.fetchMetricValues(pattern, metrics, timer)

		expected := &types.MetricData{
			FetchResponse: pb.FetchResponse{
				Name:      pattern,
				StartTime: timer.from,
				StopTime:  timer.until,
				StepTime:  60,
				Values:    []float64{},
			},
			Tags: map[string]string{"name": pattern},
		}
		So(metricValues, ShouldResemble, []*types.MetricData{expected})
		So(metrics.metrics, ShouldBeEmpty)
		So(err, ShouldBeNil)
	})

	Convey("Test one metric", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, timer.from, timer.until-1).Return(dataList, nil)

		metrics, err := fetchData.fetchMetricNames(pattern)
		So(err, ShouldBeNil)
		metricValues, err := fetchData.fetchMetricValues(pattern, metrics, timer)

		expected := &types.MetricData{
			FetchResponse: pb.FetchResponse{
				Name:      metric,
				StartTime: timer.from,
				StopTime:  timer.until,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3},
			},
			Tags: map[string]string{"name": metric},
		}
		So(metricValues, ShouldResemble, []*types.MetricData{expected})
		So(metrics.metrics, ShouldResemble, []string{metric})
		So(err, ShouldBeNil)
	})

	metric2 := "super-puper-mega-metric"
	dataList[metric2] = generateMetricValues(20, 23, 10, 5)

	Convey("Test multiple metrics", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric, metric2}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric, metric2}, timer.from, timer.until-1).Return(dataList, nil)

		metrics, err := fetchData.fetchMetricNames(pattern)
		So(err, ShouldBeNil)
		metricValues, err := fetchData.fetchMetricValues(pattern, metrics, timer)

		fetchResponse := pb.FetchResponse{
			Name:      metric,
			StartTime: timer.from,
			StopTime:  timer.until,
			StepTime:  retention,
			Values:    []float64{0, 1, 2, 3},
		}
		expected := types.MetricData{
			FetchResponse: fetchResponse,
			Tags:          map[string]string{"name": metric},
		}
		expected2 := types.MetricData{
			FetchResponse: fetchResponse,
			Tags:          map[string]string{"name": metric2},
		}
		expected2.Name = metric2

		So(metricValues, ShouldResemble, []*types.MetricData{&expected, &expected2})
		So(metrics.metrics, ShouldResemble, []string{metric, metric2})
		So(err, ShouldBeNil)
	})
}

func generateMetricValues(from, retentionFrom, retention int64, count int) []*moira.MetricValue {
	values := make([]*moira.MetricValue, 0, count)

	for i := 0; i < count; i++ {
		values = append(values, &moira.MetricValue{
			RetentionTimestamp: retentionFrom,
			Timestamp:          from,
			Value:              float64(i),
		})
		retentionFrom += retention
		from += retention
	}

	return values
}

func shouldHaveTheSameValuesAs(actual interface{}, expected ...interface{}) string {
	a := actual.([]float64)
	e := expected[0].([]float64)

	if len(a) != len(e) {
		return fmt.Sprintf("Expected '%+v', but got '%+v': different length", e, a)
	}

	for i := range a {
		if math.IsNaN(a[i]) && math.IsNaN(e[i]) || a[i] == e[i] {
			continue
		}

		return fmt.Sprintf("Expected '%+v', but got '%+v': differense at index %d", e, a, i)
	}

	return ""
}

// ************************************************************
// ************************************************************
// ************************************************************

func TestUnpackMetricValuesNoData(t *testing.T) {
	var retention int64 = 10

	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		Convey("With no metrics", func() {
			timer := NewTimerRoundingTimestamps(0, 0, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{math.NaN()}
			So(val["metric"], shouldHaveTheSameValuesAs, expected)
		})

		Convey("With one value", func() {
			value := 100.0

			metricData["metric"] = []*moira.MetricValue{
				{
					RetentionTimestamp: 0,
					Timestamp:          0,
					Value:              value,
				},
			}

			timer := NewTimerRoundingTimestamps(0, 0, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{value}
			So(val["metric"], ShouldResemble, expected)
		})
	})

	return
	Convey("Time == 9", t, func() {
		timer := NewTimerRoundingTimestamps(0, 9, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := make([]float64, 0)
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 10", t, func() {
		timer := NewTimerRoundingTimestamps(0, 10, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{100.00}
		So(val["metric"], ShouldResemble, expected)
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 20", t, func() {
		timer := NewTimerRoundingTimestamps(0, 20, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{100.00, 200.00}
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 29", t, func() {
		timer := NewTimerRoundingTimestamps(0, 29, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{100.00, 200.00}
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 30", t, func() {
		timer := NewTimerRoundingTimestamps(0, 30, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{100.00, 200.00, 300.00}
		So(val["metric"], ShouldResemble, expected)
	})
}

func TestUnpackMetricValues(t *testing.T) {
	var retention int64 = 10

	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		Convey("With one value", func() {
			value := 100.0

			metricData["metric"] = []*moira.MetricValue{
				{
					RetentionTimestamp: 0,
					Timestamp:          0,
					Value:              value,
				},
			}

			timer := NewTimerRoundingTimestamps(0, 0, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{value}
			So(val["metric"], ShouldResemble, expected)
		})
	})
}

// ************************************************************
// ************************************************************
// ************************************************************

func TestNoDataSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("AllowRealTimeAlerting == true", t, func() {
		Convey("Time == 0", func() {
			timer := NewTimerRoundingTimestamps(from, 0, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 9", func() {
			timer := NewTimerRoundingTimestamps(from, 9, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 10", func() {
			timer := NewTimerRoundingTimestamps(from, 10, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})

		Convey("Time == 11", func() {
			timer := NewTimerRoundingTimestamps(from, 11, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{math.NaN()}
			So(
				arrToString(val["metric"]),
				ShouldResemble,
				arrToString(expected),
			)
		})

		Convey("Time == 20", func() {
			timer := NewTimerRoundingTimestamps(from, 20, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{math.NaN(), math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})
	})

	Convey("AllowRealTimeAlerting == false", t, func() {
		Convey("Time == 0", func() {
			timer := NewTimerRoundingTimestamps(from, 0, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 9", func() {
			timer := NewTimerRoundingTimestamps(from, 9, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 10", func() {
			timer := NewTimerRoundingTimestamps(from, 10, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})

		Convey("Time == 11", func() {
			timer := NewTimerRoundingTimestamps(from, 11, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})

		Convey("Time == 20", func() {
			timer := NewTimerRoundingTimestamps(from, 20, retention)
			val := unpackMetricsValues(metricData, timer)
			expected := []float64{math.NaN(), math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})
	})
}

func TestConservativeMultipleSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{
		"metric1": make([]*moira.MetricValue, 0),
		"metric2": make([]*moira.MetricValue, 0),
	}

	Convey("Time == 0", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 0, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 0, retention)

		val := unpackMetricsValues(metricData, timerNotRealTime)
		So(val["metric1"], ShouldResemble, make([]float64, 0))
		So(val["metric2"], ShouldResemble, make([]float64, 0))

		val1 := unpackMetricsValues(metricData, timerRealTime)
		So(val1["metric1"], ShouldResemble, make([]float64, 0))
		So(val1["metric2"], ShouldResemble, make([]float64, 0))

		metricData["metric1"] = append(metricData["metric1"], &moira.MetricValue{Timestamp: 0, RetentionTimestamp: 0, Value: 100.00})

		val2 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val2["metric1"], ShouldResemble, make([]float64, 0))
		So(val2["metric2"], ShouldResemble, make([]float64, 0))

		val3 := unpackMetricsValues(metricData, timerRealTime)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, make([]float64, 0))
	})

	metricData["metric2"] = append(metricData["metric2"], &moira.MetricValue{Timestamp: 5, RetentionTimestamp: 5, Value: 150.00})

	Convey("Time == 5", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 5, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 5, retention)

		val1 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val1["metric1"], ShouldResemble, make([]float64, 0))
		So(val1["metric2"], ShouldResemble, make([]float64, 0))

		val3 := unpackMetricsValues(metricData, timerRealTime)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00})
	})

	Convey("Time == 9", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 9, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 9, retention)

		val1 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val1["metric1"], ShouldResemble, make([]float64, 0))
		So(val1["metric2"], ShouldResemble, make([]float64, 0))

		val3 := unpackMetricsValues(metricData, timerRealTime)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00})
	})

	Convey("Time == 10", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 10, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 10, retention)

		val1 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val1["metric1"], ShouldResemble, []float64{100.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00})

		val3 := unpackMetricsValues(metricData, timerRealTime)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00})
	})

	metricData["metric1"] = append(metricData["metric1"], &moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric2"] = append(metricData["metric2"], &moira.MetricValue{Timestamp: 15, RetentionTimestamp: 15, Value: 250.00})
	metricData["metric1"] = append(metricData["metric1"], &moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 20", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 20, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 20, retention)

		val1 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val1["metric1"], ShouldResemble, []float64{100.00, 200.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00, 250.00})

		val3 := unpackMetricsValues(metricData, timerRealTime)
		So(val3["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00, 250.00})
	})

	metricData["metric2"] = append(metricData["metric2"], &moira.MetricValue{Timestamp: 25, RetentionTimestamp: 25, Value: 350.00})

	Convey("Time == 29", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 29, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 29, retention)

		val1 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val1["metric1"], ShouldResemble, []float64{100.00, 200.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00, 250.00})

		val3 := unpackMetricsValues(metricData, timerRealTime)
		So(val3["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00, 250.00, 350.00})
	})

	Convey("Time == 30", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 30, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 30, retention)

		val1 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val1["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00, 250.00, 350.00})

		val3 := unpackMetricsValues(metricData, timerRealTime)
		So(val3["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00, 250.00, 350.00})
	})
}

func TestNonZeroStartTimeSeries(t *testing.T) {
	var retention int64 = 10
	var from int64 = 2
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 11", t, func() {
		timerNotRealTime := NewTimerRoundingTimestamps(from, 11, retention)
		timerRealTime := NewTimerRoundingTimestamps(from, 11, retention)

		val1 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
		val2 := unpackMetricsValues(metricData, timerRealTime)
		So(val2["metric"], ShouldResemble, make([]float64, 0))

		metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 11, RetentionTimestamp: 10, Value: 100.00})

		val3 := unpackMetricsValues(metricData, timerNotRealTime)
		So(val3["metric"], ShouldResemble, make([]float64, 0))
		val4 := unpackMetricsValues(metricData, timerRealTime)
		So(val4["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 12", t, func() {
		timer := NewTimerRoundingTimestamps(from, 12, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})
}

func TestConservativeShiftedSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		timer := NewTimerRoundingTimestamps(from, 0, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 4, RetentionTimestamp: 0, Value: 100.00})

	Convey("Time == 5", t, func() {
		timer := NewTimerRoundingTimestamps(from, 5, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	Convey("Time == 9", t, func() {
		timer := NewTimerRoundingTimestamps(from, 9, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	Convey("Time == 10", t, func() {
		timer := NewTimerRoundingTimestamps(from, 10, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 11", t, func() {
		timer := NewTimerRoundingTimestamps(from, 11, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 25, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 25", t, func() {
		timer := NewTimerRoundingTimestamps(from, 25, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00})
	})

	Convey("Time == 29", t, func() {
		timer := NewTimerRoundingTimestamps(from, 29, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00})
	})

	Convey("Time == 30", t, func() {
		timer := NewTimerRoundingTimestamps(from, 30, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})
}

func TestRealTimeShiftedSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		timer := NewTimerRoundingTimestamps(from, 0, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 4, RetentionTimestamp: 0, Value: 100.00})

	Convey("Time == 5", t, func() {
		timer := NewTimerRoundingTimestamps(from, 5, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 9", t, func() {
		timer := NewTimerRoundingTimestamps(from, 9, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 10", t, func() {
		timer := NewTimerRoundingTimestamps(from, 10, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 11", t, func() {
		timer := NewTimerRoundingTimestamps(from, 11, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 25, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 25", t, func() {
		timer := NewTimerRoundingTimestamps(from, 25, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})

	Convey("Time == 29", t, func() {
		timer := NewTimerRoundingTimestamps(from, 29, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})

	Convey("Time == 30", t, func() {
		timer := NewTimerRoundingTimestamps(from, 30, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})
}

func arrToString(arr []float64) string {
	return fmt.Sprintf("%v", arr)
}
