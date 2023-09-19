package prometheus

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	metricsource "github.com/moira-alert/moira/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/prometheus/common/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPrometheusFetch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	api := mock_moira_alert.NewMockPrometheusApi(ctrl)

	logger, _ := zerolog_adapter.GetLogger("Test")

	now := time.Now()

	prometheus := Prometheus{
		config: &Config{},
		api:    api,
		logger: logger,
	}

	Convey("Given two metric points", t, func() {
		fromMilli := now.UnixMilli()
		untilMilli := now.Add(time.Second * 60).UnixMilli()

		from := now.Unix()
		until := now.Add(time.Second * 60).Unix()

		api.EXPECT().QueryRange(gomock.Any(), "target", gomock.Any()).
			Return(
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{"__name__": "name1"},
						Values: []model.SamplePair{
							{Timestamp: model.Time(fromMilli), Value: 3.14},
							{Timestamp: model.Time(untilMilli), Value: 2.71},
						},
					},
				},
				nil,
				nil,
			)

		res, err := prometheus.fetch("target", fromMilli, untilMilli, true)

		So(err, ShouldBeNil)
		So(res, ShouldResemble, &FetchResult{
			MetricsData: []metricsource.MetricData{
				{
					Name:      "name1",
					StartTime: from,
					StopTime:  until,
					StepTime:  60,
					Values:    []float64{3.14, 2.71},
					Wildcard:  false,
				},
			},
		})
	})
}

func TestPrometheusFetchRetries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	api := mock_moira_alert.NewMockPrometheusApi(ctrl)

	logger, _ := zerolog_adapter.GetLogger("Test")

	now := time.Now()

	prometheus := Prometheus{
		config: &Config{Retries: 3, RetryTimeout: time.Second * 0},
		api:    api,
		logger: logger,
	}

	Convey("Given two metric points and two fails", t, func() {
		fromMilli := now.UnixMilli()
		untilMilli := now.Add(time.Second * 60).UnixMilli()

		from := now.Unix()
		until := now.Add(time.Second * 60).Unix()

		api.EXPECT().QueryRange(gomock.Any(), "target", gomock.Any()).
			Return(nil, nil, fmt.Errorf("Error")).Times(2)
		api.EXPECT().QueryRange(gomock.Any(), "target", gomock.Any()).
			Return(
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{"__name__": "name1"},
						Values: []model.SamplePair{
							{Timestamp: model.Time(fromMilli), Value: 3.14},
							{Timestamp: model.Time(untilMilli), Value: 2.71},
						},
					},
				},
				nil,
				nil,
			)

		res, err := prometheus.Fetch("target", fromMilli, untilMilli, true)

		So(err, ShouldBeNil)
		So(res, ShouldResemble, &FetchResult{
			MetricsData: []metricsource.MetricData{
				{
					Name:      "name1",
					StartTime: from,
					StopTime:  until,
					StepTime:  60,
					Values:    []float64{3.14, 2.71},
					Wildcard:  false,
				},
			},
		})
	})

	Convey("Given all requests failed", t, func() {
		fromMilli := now.UnixMilli()
		untilMilli := now.Add(time.Second * 60).UnixMilli()

		expectedErr := fmt.Errorf("Error")

		api.EXPECT().QueryRange(gomock.Any(), "target", gomock.Any()).
			Return(nil, nil, expectedErr).Times(3)

		res, err := prometheus.Fetch("target", fromMilli, untilMilli, true)

		So(res, ShouldBeNil)
		So(err, ShouldEqual, expectedErr)
	})
}
