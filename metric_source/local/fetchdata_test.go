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

	metricValues := []*moira.MetricValue{
		{RetentionTimestamp: 20, Timestamp: 23, Value: 0},
		{RetentionTimestamp: 30, Timestamp: 33, Value: 1},
		{RetentionTimestamp: 40, Timestamp: 43, Value: 2},
		{RetentionTimestamp: 50, Timestamp: 53, Value: 3},
		{RetentionTimestamp: 60, Timestamp: 63, Value: 4},
	}

	dataList := map[string][]*moira.MetricValue{
		metric: metricValues,
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
	dataList[metric2] = metricValues

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

func TestUnpackMetricValuesNoData(t *testing.T) {
	var retention int64 = 10

	metricData := map[string][]*moira.MetricValue{"metric": make([]*moira.MetricValue, 0)}

	Convey("From 1 until 1", t, func() {
		timer := NewTimerRoundingTimestamps(1, 1, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{}
		So(val["metric"], shouldHaveTheSameValuesAs, expected)
	})

	Convey("From 0 until 0", t, func() {
		timer := NewTimerRoundingTimestamps(0, 0, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{math.NaN()}
		So(val["metric"], shouldHaveTheSameValuesAs, expected)
	})

	Convey("From 0 until 10", t, func() {
		timer := NewTimerRoundingTimestamps(0, 10, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{math.NaN(), math.NaN()}
		So(val["metric"], shouldHaveTheSameValuesAs, expected)
	})

	Convey("From 1 until 11", t, func() {
		timer := NewTimerRoundingTimestamps(1, 11, retention)
		val := unpackMetricsValues(metricData, timer)
		expected := []float64{math.NaN()}
		So(val["metric"], shouldHaveTheSameValuesAs, expected)
	})
}

func TestUnpackMetricValues(t *testing.T) {
	var retention int64 = 10

	metricData := map[string][]*moira.MetricValue{"metric": {
		{Timestamp: 0, RetentionTimestamp: 0, Value: 100.00},
		{Timestamp: 10, RetentionTimestamp: 10, Value: 200.00},
		{Timestamp: 20, RetentionTimestamp: 20, Value: 300.00},
	}}

	Convey("From 1 until 1", t, func() {
		timer := NewTimerRoundingTimestamps(1, 1, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{})
	})

	Convey("From 0 until 0", t, func() {
		timer := NewTimerRoundingTimestamps(0, 0, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.0})
	})

	Convey("From 1 until 11", t, func() {
		timer := NewTimerRoundingTimestamps(1, 11, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{200.0})
	})

	Convey("From 0 until 10", t, func() {
		timer := NewTimerRoundingTimestamps(0, 10, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.00, 200.0})
	})

	Convey("From 0 until 11", t, func() {
		timer := NewTimerRoundingTimestamps(0, 11, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.00, 200.00})
	})

	Convey("From 0 until 19", t, func() {
		timer := NewTimerRoundingTimestamps(0, 19, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.00, 200.00})
	})

	Convey("From 1 until 30", t, func() {
		timer := NewTimerRoundingTimestamps(1, 30, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{200.00, 300.00, math.NaN()})
	})
}

func TestMultipleSeriesNoData(t *testing.T) {
	var retention int64 = 10
	metricData := map[string][]*moira.MetricValue{
		"metric1": {},
		"metric2": {},
	}

	Convey("From 1 until 1", t, func() {
		timer := NewTimerRoundingTimestamps(1, 1, retention)

		val := unpackMetricsValues(metricData, timer)
		So(val["metric1"], shouldHaveTheSameValuesAs, []float64{})
		So(val["metric2"], shouldHaveTheSameValuesAs, []float64{})
	})

	Convey("From 0 until 0", t, func() {
		timer := NewTimerRoundingTimestamps(0, 0, retention)

		val := unpackMetricsValues(metricData, timer)
		So(val["metric1"], shouldHaveTheSameValuesAs, []float64{math.NaN()})
		So(val["metric2"], shouldHaveTheSameValuesAs, []float64{math.NaN()})
	})

	Convey("From 1 until 5", t, func() {
		timer := NewTimerRoundingTimestamps(1, 5, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{})
	})

	Convey("From 0 until 5", t, func() {
		timer := NewTimerRoundingTimestamps(0, 5, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{math.NaN()})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{math.NaN()})
	})

	Convey("From 5 until 30", t, func() {
		timer := NewTimerRoundingTimestamps(5, 30, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{math.NaN(), math.NaN(), math.NaN()})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{math.NaN(), math.NaN(), math.NaN()})
	})
}

func TestMultipleSeries(t *testing.T) {
	var retention int64 = 10

	metricData := map[string][]*moira.MetricValue{
		"metric1": {
			{Timestamp: 0, RetentionTimestamp: 0, Value: 100.00},
			{Timestamp: 10, RetentionTimestamp: 10, Value: 200.00},
			{Timestamp: 20, RetentionTimestamp: 20, Value: 300.00},
		},
		"metric2": {
			{Timestamp: 0, RetentionTimestamp: 0, Value: 150.00},
			{Timestamp: 10, RetentionTimestamp: 10, Value: 250.00},
			{Timestamp: 20, RetentionTimestamp: 20, Value: 350.00},
		},
	}

	Convey("From 1 until 1", t, func() {
		timer := NewTimerRoundingTimestamps(1, 1, retention)

		val := unpackMetricsValues(metricData, timer)
		So(val["metric1"], shouldHaveTheSameValuesAs, []float64{})
		So(val["metric2"], shouldHaveTheSameValuesAs, []float64{})
	})

	Convey("From 0 until 0", t, func() {
		timer := NewTimerRoundingTimestamps(0, 0, retention)

		val := unpackMetricsValues(metricData, timer)
		So(val["metric1"], shouldHaveTheSameValuesAs, []float64{100.0})
		So(val["metric2"], shouldHaveTheSameValuesAs, []float64{150.0})
	})

	Convey("From 1 until 5", t, func() {
		timer := NewTimerRoundingTimestamps(1, 5, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{})
	})

	Convey("From 0 until 5", t, func() {
		timer := NewTimerRoundingTimestamps(0, 5, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{100.0})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{150.0})
	})

	Convey("From 0 until 9", t, func() {
		timer := NewTimerRoundingTimestamps(0, 9, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{100.00})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{150.00})
	})

	Convey("From 0 until 10", t, func() {
		timer := NewTimerRoundingTimestamps(0, 10, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{100.00, 200.00})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{150.00, 250.00})
	})

	Convey("From 1 until 11", t, func() {
		timer := NewTimerRoundingTimestamps(1, 11, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{200.00})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{250.00})
	})

	Convey("From 0 until 30", t, func() {
		timer := NewTimerRoundingTimestamps(0, 30, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{100.00, 200.00, 300.00, math.NaN()})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{150.00, 250.00, 350.00, math.NaN()})
	})

	Convey("From 5 until 30", t, func() {
		timer := NewTimerRoundingTimestamps(5, 30, retention)

		val1 := unpackMetricsValues(metricData, timer)
		So(val1["metric1"], shouldHaveTheSameValuesAs, []float64{200.00, 300.00, math.NaN()})
		So(val1["metric2"], shouldHaveTheSameValuesAs, []float64{250.00, 350.00, math.NaN()})
	})
}

func TestShiftedSeries(t *testing.T) {
	var retention int64 = 10
	metricData := map[string][]*moira.MetricValue{"metric": {
		{Timestamp: 4, RetentionTimestamp: 0, Value: 100.00},
		{Timestamp: 15, RetentionTimestamp: 10, Value: 200.00},
		{Timestamp: 25, RetentionTimestamp: 20, Value: 300.00},
	}}

	Convey("From 1 until 1", t, func() {
		timer := NewTimerRoundingTimestamps(1, 1, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{})
	})

	Convey("From 0 until 0", t, func() {
		timer := NewTimerRoundingTimestamps(0, 0, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.0})
	})

	Convey("From 1 until 11", t, func() {
		timer := NewTimerRoundingTimestamps(1, 11, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{200.0})
	})

	Convey("From 0 until 10", t, func() {
		timer := NewTimerRoundingTimestamps(0, 10, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.00, 200.0})
	})

	Convey("From 0 until 11", t, func() {
		timer := NewTimerRoundingTimestamps(0, 11, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.00, 200.00})
	})

	Convey("From 0 until 19", t, func() {
		timer := NewTimerRoundingTimestamps(0, 19, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{100.00, 200.00})
	})

	Convey("From 1 until 30", t, func() {
		timer := NewTimerRoundingTimestamps(1, 30, retention)
		val := unpackMetricsValues(metricData, timer)

		So(val["metric"], shouldHaveTheSameValuesAs, []float64{200.00, 300.00, math.NaN()})
	})
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
