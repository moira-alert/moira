package prometheus

import (
	"testing"
	"time"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/prometheus/common/model"
	. "github.com/smartystreets/goconvey/convey"
)

func MakeSamplePair(time time.Time, value float64) model.SamplePair {
	return model.SamplePair{
		Timestamp: model.Time(time.UnixMilli()),
		Value:     model.SampleValue(value),
	}
}

func TestConvertToFetchResult(t *testing.T) {
	metric_1 := model.Metric{
		"__name__": "name",
		"label_1":  "value_1",
	}
	metric_2 := model.Metric{
		"__name__": "name",
		"label_1":  "value_2",
	}

	Convey("Given no metrics fetched", t, func() {
		now := time.Now()

		mat := model.Matrix{}

		result := convertToFetchResult(mat, now.Unix(), now.Unix())

		expected := &FetchResult{
			MetricsData: make([]metricSource.MetricData, 0),
		}

		So(result, ShouldResemble, expected)
	})

	Convey("Given one metric with one value fetched", t, func() {
		now := time.Now()

		mat := model.Matrix{&model.SampleStream{
			Metric: metric_1,
			Values: []model.SamplePair{MakeSamplePair(now, 1.0)},
		}}

		result := convertToFetchResult(mat, now.Unix(), now.Unix())

		expected := &FetchResult{
			MetricsData: []metricSource.MetricData{
				{
					Name:      "name;label_1=value_1",
					StartTime: now.Unix(),
					StopTime:  now.Unix(),
					StepTime:  60,
					Values:    []float64{1.0},
					Wildcard:  false,
				},
			},
		}

		So(result, ShouldResemble, expected)
	})

	Convey("Given one metric with many values fetched", t, func() {
		now := time.Now()

		mat := model.Matrix{&model.SampleStream{
			Metric: metric_1,
			Values: []model.SamplePair{
				MakeSamplePair(now, 1.0),
				MakeSamplePair(now.Add(60*time.Second), 2.0),
				MakeSamplePair(now.Add(120*time.Second), 3.0),
				MakeSamplePair(now.Add(180*time.Second), 4.0),
			},
		}}

		result := convertToFetchResult(mat, now.Unix(), now.Unix())

		expected := &FetchResult{
			MetricsData: []metricSource.MetricData{
				{
					Name:      "name;label_1=value_1",
					StartTime: now.Unix(),
					StopTime:  now.Add(180 * time.Second).Unix(),
					StepTime:  60,
					Values:    []float64{1.0, 2.0, 3.0, 4.0},
					Wildcard:  false,
				},
			},
		}

		So(result, ShouldResemble, expected)
	})

	Convey("Given several metric fetched", t, func() {
		now := time.Now()

		mat := model.Matrix{
			&model.SampleStream{
				Metric: metric_1,
				Values: []model.SamplePair{
					MakeSamplePair(now, 1.0),
					MakeSamplePair(now.Add(60*time.Second), 2.0),
				},
			},
			&model.SampleStream{
				Metric: metric_2,
				Values: []model.SamplePair{
					MakeSamplePair(now, 3.0),
					MakeSamplePair(now.Add(60*time.Second), 4.0),
				},
			},
		}

		result := convertToFetchResult(mat, now.Unix(), now.Unix())

		expected := &FetchResult{
			MetricsData: []metricSource.MetricData{
				{
					Name:      "name;label_1=value_1",
					StartTime: now.Unix(),
					StopTime:  now.Add(60 * time.Second).Unix(),
					StepTime:  60,
					Values:    []float64{1.0, 2.0},
					Wildcard:  false,
				},
				{
					Name:      "name;label_1=value_2",
					StartTime: now.Unix(),
					StopTime:  now.Add(60 * time.Second).Unix(),
					StepTime:  60,
					Values:    []float64{3.0, 4.0},
					Wildcard:  false,
				},
			},
		}

		So(result, ShouldResemble, expected)
	})
}

func TestTargetFromTags(t *testing.T) {
	Convey("Given tags is empty", t, func() {
		tags := model.Metric{}

		target := targetFromTags(tags)

		So(target, ShouldEqual, "")
	})

	Convey("Given tags contain only __name__", t, func() {
		tags := model.Metric{
			"__name__": "name",
		}

		target := targetFromTags(tags)

		So(target, ShouldEqual, "name")
	})

	Convey("Given tags contain one tag", t, func() {
		tags := model.Metric{
			"test_1": "value_1",
		}

		target := targetFromTags(tags)

		So(target, ShouldEqual, "test_1=value_1")
	})

	Convey("Given tags contain several tags", t, func() {
		tags := model.Metric{
			"a":  "1",
			"ab": "1",
			"c":  "1",
		}

		target := targetFromTags(tags)

		So(target, ShouldEqual, "a=1;ab=1;c=1")
	})

	Convey("Given tags contain __name__ and tags", t, func() {
		tags := model.Metric{
			"test_1":   "value_1",
			"__name__": "name",
			"test_2":   "value_2",
		}

		target := targetFromTags(tags)

		So(target, ShouldEqual, "name;test_1=value_1;test_2=value_2")
	})
}
