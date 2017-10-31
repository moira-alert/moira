package target

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"math"
	"testing"
)

func TestFetchData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	pattern := "super-puper-pattern"
	metric := "super-puper-metric"
	dataList := map[string][]*moira.MetricValue{
		metric: {
			{
				RetentionTimestamp: 20,
				Timestamp:          23,
				Value:              0,
			},
			{
				RetentionTimestamp: 30,
				Timestamp:          33,
				Value:              1,
			},
			{
				RetentionTimestamp: 40,
				Timestamp:          43,
				Value:              2,
			},
			{
				RetentionTimestamp: 50,
				Timestamp:          53,
				Value:              3,
			},
			{
				RetentionTimestamp: 60,
				Timestamp:          63,
				Value:              4,
			},
		},
	}

	var from int64 = 17
	var until int64 = 67
	var retention int64 = 10
	retentionErr := fmt.Errorf("Ooops, retention error")
	patternErr := fmt.Errorf("Ooops, pattern error")
	metricErr := fmt.Errorf("Ooops, metric error")

	Convey("Errors Test", t, func() {
		Convey("GetPatternMetricsError", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return(nil, patternErr)
			metricData, metrics, err := FetchData(dataBase, pattern, from, until, true)
			So(metricData, ShouldBeNil)
			So(metrics, ShouldBeNil)
			So(err, ShouldResemble, patternErr)
		})
		Convey("GetMetricRetentionError", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(int64(0), retentionErr)
			metricData, metrics, err := FetchData(dataBase, pattern, from, until, true)
			So(metricData, ShouldBeNil)
			So(metrics, ShouldBeNil)
			So(err, ShouldResemble, retentionErr)
		})
		Convey("GetMetricsValuesError", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(nil, metricErr)
			metricData, metrics, err := FetchData(dataBase, pattern, from, until, true)
			So(metricData, ShouldBeNil)
			So(metrics, ShouldBeNil)
			So(err, ShouldResemble, metricErr)
		})
	})

	Convey("Test no metrics", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
		metricData, metrics, err := FetchData(dataBase, pattern, from, until, false)
		fetchResponse := pb.FetchResponse{
			Name:      pattern,
			StartTime: int32(from),
			StopTime:  int32(until),
			StepTime:  60,
			Values:    []float64{},
			IsAbsent:  []bool{},
		}
		expected := &expr.MetricData{FetchResponse: fetchResponse}
		So(metricData, ShouldResemble, []*expr.MetricData{expected})
		So(metrics, ShouldBeEmpty)
		So(err, ShouldBeNil)
	})

	Convey("Test allowRealTimeAlerting=true", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
		metricData, metrics, err := FetchData(dataBase, pattern, from, until, false)
		fetchResponse := pb.FetchResponse{
			Name:      metric,
			StartTime: int32(from),
			StopTime:  int32(until),
			StepTime:  int32(retention),
			Values:    []float64{0, 1, 2, 3},
			IsAbsent:  make([]bool, 4),
		}
		expected := &expr.MetricData{FetchResponse: fetchResponse}
		So(metricData, ShouldResemble, []*expr.MetricData{expected})
		So(metrics, ShouldResemble, []string{metric})
		So(err, ShouldBeNil)
	})

	Convey("Test allowRealTimeAlerting=true", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
		metricData, metrics, err := FetchData(dataBase, pattern, from, until, true)
		fetchResponse := pb.FetchResponse{
			Name:      metric,
			StartTime: int32(from),
			StopTime:  int32(until),
			StepTime:  int32(retention),
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  make([]bool, 5),
		}
		expected := &expr.MetricData{FetchResponse: fetchResponse}
		So(metricData, ShouldResemble, []*expr.MetricData{expected})
		So(metrics, ShouldResemble, []string{metric})
		So(err, ShouldBeNil)
	})

	metric2 := "super-puper-mega-metric"
	dataList[metric2] = []*moira.MetricValue{
		{
			RetentionTimestamp: 20,
			Timestamp:          23,
			Value:              0,
		},
		{
			RetentionTimestamp: 30,
			Timestamp:          33,
			Value:              1,
		},
		{
			RetentionTimestamp: 40,
			Timestamp:          43,
			Value:              2,
		},
		{
			RetentionTimestamp: 50,
			Timestamp:          53,
			Value:              3,
		},
		{
			RetentionTimestamp: 60,
			Timestamp:          63,
			Value:              4,
		},
	}

	Convey("Test multiple metrics", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric, metric2}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric, metric2}, from, until).Return(dataList, nil)
		metricData, metrics, err := FetchData(dataBase, pattern, from, until, true)
		fetchResponse := pb.FetchResponse{
			Name:      metric,
			StartTime: int32(from),
			StopTime:  int32(until),
			StepTime:  int32(retention),
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  make([]bool, 5),
		}
		expected := expr.MetricData{FetchResponse: fetchResponse}
		expected2 := expected
		expected2.Name = metric2
		So(metricData, ShouldResemble, []*expr.MetricData{&expected, &expected2})
		So(metrics, ShouldResemble, []string{metric, metric2})
		So(err, ShouldBeNil)
	})

	mockCtrl.Finish()
}

func TestAllowRealTimeAlerting(t *testing.T) {
	metricsValues := []*moira.MetricValue{
		{
			RetentionTimestamp: 20,
			Timestamp:          23,
			Value:              0,
		},
		{
			RetentionTimestamp: 30,
			Timestamp:          33,
			Value:              1,
		},
		{
			RetentionTimestamp: 40,
			Timestamp:          43,
			Value:              2,
		},
		{
			RetentionTimestamp: 50,
			Timestamp:          53,
			Value:              3,
		},
		{
			RetentionTimestamp: 60,
			Timestamp:          63,
			Value:              4,
		},
	}

	metricData := map[string][]*moira.MetricValue{"metric1": metricsValues}

	Convey("Test full interval", t, func() {
		Convey("AllowRealTimeAlerting is false, should be truncated on left", func() {
			val := unpackMetricsValues(metricData, 10, 17, 67, false)
			expected := []float64{0, 1, 2, 3}
			So(val["metric1"], ShouldResemble, expected)
		})

		Convey("AllowRealTimeAlerting is true, should full interval", func() {
			val := unpackMetricsValues(metricData, 10, 17, 67, true)
			expected := []float64{0, 1, 2, 3, 4}
			So(val["metric1"], ShouldResemble, expected)
		})
	})

	Convey("Test interval truncated on the right", t, func() {
		Convey("AllowRealTimeAlerting is false, should be truncated on left and right", func() {
			val := unpackMetricsValues(metricData, 10, 24, 67, false)
			expected := []float64{1, 2, 3}
			So(val["metric1"], ShouldResemble, expected)
		})

		Convey("AllowRealTimeAlerting is true, should be truncated on the right", func() {
			val := unpackMetricsValues(metricData, 10, 24, 67, true)
			expected := []float64{1, 2, 3, 4}
			So(val["metric1"], ShouldResemble, expected)
		})
	})
}

func TestConservativeAlignedSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		Convey("No Metric Values", func() {
			val := unpackMetricsValues(metricData, retention, from, 0, false)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})
		Convey("Has Metric Values", func() {
			metricData["metric"] = []*moira.MetricValue{
				{
					RetentionTimestamp: 0,
					Timestamp:          0,
					Value:              100.00,
				},
			}
			val := unpackMetricsValues(metricData, retention, from, 0, false)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})
	})

	Convey("Time == 9", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 9, false)
		expected := make([]float64, 0)
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 10", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 10, false)
		expected := []float64{100.00}
		So(val["metric"], ShouldResemble, expected)
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 20", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 20, false)
		expected := []float64{100.00, 200.00}
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 29", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 29, false)
		expected := []float64{100.00, 200.00}
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 30", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 30, false)
		expected := []float64{100.00, 200.00, 300.00}
		So(val["metric"], ShouldResemble, expected)
	})
}

func TestRealTimeAlignedSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		Convey("No Metric Values", func() {
			val := unpackMetricsValues(metricData, retention, from, 0, true)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})
		Convey("Has Metric Values", func() {
			metricData["metric"] = []*moira.MetricValue{
				{
					RetentionTimestamp: 0,
					Timestamp:          0,
					Value:              100.00,
				},
			}
			val := unpackMetricsValues(metricData, retention, from, 0, true)
			expected := []float64{100.00}
			So(val["metric"], ShouldResemble, expected)
		})
	})

	Convey("Time == 9", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 9, true)
		expected := []float64{100.00}
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 10", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 10, true)
		expected := []float64{100.00}
		So(val["metric"], ShouldResemble, expected)
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 20", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 20, true)
		expected := []float64{100.00, 200.00, 300.00}
		So(val["metric"], ShouldResemble, expected)
	})
	Convey("Time == 29", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 29, true)
		expected := []float64{100.00, 200.00, 300.00}
		So(val["metric"], ShouldResemble, expected)
	})

	Convey("Time == 30", t, func() {
		val := unpackMetricsValues(metricData, retention, from, 30, true)
		expected := []float64{100.00, 200.00, 300.00}
		So(val["metric"], ShouldResemble, expected)
	})
}

func TestNoDataSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("AllowRealTimeAlerting == true", t, func() {
		Convey("Time == 0", func() {
			val := unpackMetricsValues(metricData, retention, from, 0, true)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 9", func() {
			val := unpackMetricsValues(metricData, retention, from, 9, true)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 10", func() {
			val := unpackMetricsValues(metricData, retention, from, 10, true)
			expected := []float64{math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})

		Convey("Time == 11", func() {
			val := unpackMetricsValues(metricData, retention, from, 11, true)
			expected := []float64{math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})

		Convey("Time == 20", func() {
			val := unpackMetricsValues(metricData, retention, from, 20, true)
			expected := []float64{math.NaN(), math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})
	})

	Convey("AllowRealTimeAlerting == false", t, func() {
		Convey("Time == 0", func() {
			val := unpackMetricsValues(metricData, retention, from, 0, false)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 9", func() {
			val := unpackMetricsValues(metricData, retention, from, 9, true)
			expected := make([]float64, 0)
			So(val["metric"], ShouldResemble, expected)
		})

		Convey("Time == 10", func() {
			val := unpackMetricsValues(metricData, retention, from, 10, false)
			expected := []float64{math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})

		Convey("Time == 11", func() {
			val := unpackMetricsValues(metricData, retention, from, 11, false)
			expected := []float64{math.NaN()}
			So(arrToString(val["metric"]), ShouldResemble, arrToString(expected))
		})

		Convey("Time == 20", func() {
			val := unpackMetricsValues(metricData, retention, from, 20, false)
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
		val := unpackMetricsValues(metricData, retention, from, 0, false)
		So(val["metric1"], ShouldResemble, make([]float64, 0))
		So(val["metric2"], ShouldResemble, make([]float64, 0))

		val1 := unpackMetricsValues(metricData, retention, from, 0, true)
		So(val1["metric1"], ShouldResemble, make([]float64, 0))
		So(val1["metric2"], ShouldResemble, make([]float64, 0))

		metricData["metric1"] = append(metricData["metric1"], &moira.MetricValue{Timestamp: 0, RetentionTimestamp: 0, Value: 100.00})

		val2 := unpackMetricsValues(metricData, retention, from, 0, false)
		So(val2["metric1"], ShouldResemble, make([]float64, 0))
		So(val2["metric2"], ShouldResemble, make([]float64, 0))

		val3 := unpackMetricsValues(metricData, retention, from, 0, true)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, make([]float64, 0))
	})

	metricData["metric2"] = append(metricData["metric2"], &moira.MetricValue{Timestamp: 5, RetentionTimestamp: 5, Value: 150.00})

	Convey("Time == 5", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 5, false)
		So(val1["metric1"], ShouldResemble, make([]float64, 0))
		So(val1["metric2"], ShouldResemble, make([]float64, 0))

		val3 := unpackMetricsValues(metricData, retention, from, 5, true)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00})
	})

	Convey("Time == 9", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 9, false)
		So(val1["metric1"], ShouldResemble, make([]float64, 0))
		So(val1["metric2"], ShouldResemble, make([]float64, 0))

		val3 := unpackMetricsValues(metricData, retention, from, 9, true)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00})
	})

	Convey("Time == 10", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 10, false)
		So(val1["metric1"], ShouldResemble, []float64{100.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00})

		val3 := unpackMetricsValues(metricData, retention, from, 10, true)
		So(val3["metric1"], ShouldResemble, []float64{100.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00})
	})

	metricData["metric1"] = append(metricData["metric1"], &moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric2"] = append(metricData["metric2"], &moira.MetricValue{Timestamp: 15, RetentionTimestamp: 15, Value: 250.00})
	metricData["metric1"] = append(metricData["metric1"], &moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 20", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 20, false)
		So(val1["metric1"], ShouldResemble, []float64{100.00, 200.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00, 250.00})

		val3 := unpackMetricsValues(metricData, retention, from, 20, true)
		So(val3["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00, 250.00})
	})

	metricData["metric2"] = append(metricData["metric2"], &moira.MetricValue{Timestamp: 25, RetentionTimestamp: 25, Value: 350.00})

	Convey("Time == 29", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 29, false)
		So(val1["metric1"], ShouldResemble, []float64{100.00, 200.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00, 250.00})

		val3 := unpackMetricsValues(metricData, retention, from, 29, true)
		So(val3["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00, 250.00, 350.00})
	})

	Convey("Time == 30", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 30, false)
		So(val1["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val1["metric2"], ShouldResemble, []float64{150.00, 250.00, 350.00})

		val3 := unpackMetricsValues(metricData, retention, from, 30, true)
		So(val3["metric1"], ShouldResemble, []float64{100.00, 200.00, 300.00})
		So(val3["metric2"], ShouldResemble, []float64{150.00, 250.00, 350.00})
	})
}

func TestNonZeroStartTimeSeries(t *testing.T) {
	var retention int64 = 10
	var from int64 = 2
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 11", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 11, false)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
		val2 := unpackMetricsValues(metricData, retention, from, 11, true)
		So(val2["metric"], ShouldResemble, make([]float64, 0))

		metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 11, RetentionTimestamp: 10, Value: 100.00})

		val3 := unpackMetricsValues(metricData, retention, from, 11, false)
		So(val3["metric"], ShouldResemble, make([]float64, 0))
		val4 := unpackMetricsValues(metricData, retention, from, 11, true)
		So(val4["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 12", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 12, false)
		So(val1["metric"], ShouldResemble, make([]float64, 0))

		val2 := unpackMetricsValues(metricData, retention, from, 12, true)
		So(val2["metric"], ShouldResemble, []float64{100.00})
	})
}

func TestConservativeShiftedSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 0, false)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 4, RetentionTimestamp: 0, Value: 100.00})

	Convey("Time == 5", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 5, false)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	Convey("Time == 9", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 9, false)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	Convey("Time == 10", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 10, false)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 11", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 11, false)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 25, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 25", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 25, false)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00})
	})

	Convey("Time == 29", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 29, false)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00})
	})

	Convey("Time == 30", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 30, false)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})
}

func TestRealTimeShiftedSeries(t *testing.T) {
	var retention int64 = 10
	var from int64
	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("Time == 0", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 0, true)
		So(val1["metric"], ShouldResemble, make([]float64, 0))
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 4, RetentionTimestamp: 0, Value: 100.00})

	Convey("Time == 5", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 5, true)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 9", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 9, true)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 10", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 10, true)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	Convey("Time == 11", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 11, true)
		So(val1["metric"], ShouldResemble, []float64{100.00})
	})

	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 200.00})
	metricData["metric"] = append(metricData["metric"], &moira.MetricValue{Timestamp: 25, RetentionTimestamp: 20, Value: 300.00})

	Convey("Time == 25", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 25, true)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})

	Convey("Time == 29", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 29, true)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})

	Convey("Time == 30", t, func() {
		val1 := unpackMetricsValues(metricData, retention, from, 30, true)
		So(val1["metric"], ShouldResemble, []float64{100.00, 200.00, 300.00})
	})
}

func arrToString(arr []float64) string {
	return fmt.Sprintf("%v", arr)
}
