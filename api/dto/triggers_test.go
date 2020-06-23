package dto

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/middleware"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerValidation(t *testing.T) {
	Convey("Tests targets, values and expression validation", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
		sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)

		request, _ := http.NewRequest("PUT", "/api/trigger", nil)
		request.Header.Set("Content-Type", "application/json")
		ctx := request.Context()
		ctx = context.WithValue(ctx, middleware.ContextKey("metricSourceProvider"), sourceProvider)
		request = request.WithContext(ctx)

		desc := "Graphite ClickHouse"
		tags := []string{"Normal", "DevOps", "DevOpsGraphite-duty"}
		throttling := int64(0)
		warnValue := float64(10)
		errorValue := float64(5)

		trigger := TriggerModel{
			ID:             "GraphiteStoragesFreeSpace",
			Name:           "Graphite storage free space low",
			Desc:           &desc,
			Tags:           tags,
			TTLState:       &moira.TTLStateNODATA,
			TTL:            600,
			IsRemote:       false,
			MuteNewMetrics: false,
		}

		Convey("Test FallingTrigger", func() {

			localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.TriggerType = moira.FallingTrigger

			Convey("and one target", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				Convey("and expression", func() {
					trigger.Expression = "(t1 < 10 && t2 < 10) ? WARN:OK"
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'expression' to trigger_type: 'falling'")})
				})

				Convey("and warn_value and error_value", func() {
					trigger.WarnValue = &warnValue
					trigger.ErrorValue = &errorValue
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldBeNil)
				})
			})

			Convey("and one multiple targets", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				trigger.WarnValue = &warnValue
				trigger.ErrorValue = &errorValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use trigger_type not 'falling' for with multiple targets")})
			})

		})
		Convey("Test RisingTrigger", func() {

			localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.TriggerType = moira.RisingTrigger

			Convey("and one target", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				Convey("and expression", func() {
					trigger.Expression = "(t1 < 10 && t2 < 10) ? WARN:OK"
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'expression' to trigger_type: 'rising'")})
				})

				Convey("and warn_value and error_value", func() {
					trigger.WarnValue = &errorValue
					trigger.ErrorValue = &warnValue
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldBeNil)
				})
			})

			Convey("and one multiple targets", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				trigger.WarnValue = &errorValue
				trigger.ErrorValue = &warnValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use trigger_type not 'rising' for with multiple targets")})
			})
		})
		Convey("Test ExpressionTrigger", func() {

			localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.TriggerType = moira.ExpressionTrigger
			trigger.Expression = "(t1 < 10 && t2 < 10) ? WARN:OK"
			trigger.Targets = []string{
				"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				"aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				"aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
				"aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
			}
			Convey("and warn_value", func() {
				trigger.WarnValue = &warnValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldNotBeNil)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'warn_value' on trigger_type: 'expression'")})
			})

			Convey("and error_value", func() {
				trigger.ErrorValue = &errorValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'error_value' on trigger_type: 'expression'")})
			})
			Convey("and expression", func() {
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
		})

		Convey("Test alone metrics", func() {
			localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.Targets = []string{"test target"}
			trigger.Expression = "OK"
			Convey("are empty", func() {
				trigger.AloneMetrics = map[string]bool{}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
			Convey("are valid", func() {
				trigger.AloneMetrics = map[string]bool{"t1": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
			Convey("have invalid metric name", func() {
				trigger.AloneMetrics = map[string]bool{"ttt": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("alone metrics target name should be in pattern: t\\d+")})
			})
			Convey("have target higher than total amount of targets", func() {
				trigger.AloneMetrics = map[string]bool{"t3": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("alone metrics target index should be in range from 1 to length of targets")})
			})
		})

		Convey("Test patterns", func() {
			localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.Expression = "OK"
			Convey("do not have asterisk", func() {
				trigger.Targets = []string{"sumSeries(some.test.series.*)"}
				tr := Trigger{trigger, throttling}
				fetchResult.EXPECT().GetPatterns().Return([]string{"some.test.series.*"}, nil).AnyTimes()
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
			Convey("have asterisk", func() {
				trigger.Targets = []string{"sumSeries(*)"}
				tr := Trigger{trigger, throttling}
				fetchResult.EXPECT().GetPatterns().Return([]string{"*"}, nil).AnyTimes()
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("pattern \"*\" is not allowed to use")})
			})
		})
	})
}
