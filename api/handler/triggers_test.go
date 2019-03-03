package handler

import (
	"bytes"
	"context"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/mock/metric_source"
	"net/http"
	"testing"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api/dto"
	. "github.com/smartystreets/goconvey/convey"
)


func TestExpressionModeMultipleTargetsWarnValue (t *testing.T){
	var body bytes.Buffer
	body.WriteString(`{
  "id": "GraphiteStoragesFreeSpace",
  "name": "Graphite storage free space low",
  "desc": "Graphite ClickHouse",
  "targets": [
    "aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)"
  ],
  "warn_value": 10,
  "trigger_type": "expression",
  "tags": [
    "Normal",
    "DevOps",
    "DevOpsGraphite-duty"
  ],
  "ttl_state": "NODATA",
  "ttl": 600,
  "expression": "(t1 < 10 && t2 < 10) ? WARN:OK",
  "is_remote": false,
  "mute_new_metrics": false,
  "throttling": 0
	}`)

	trigger := &dto.Trigger{}

	request, _ := http.NewRequest("PUT", "/api/trigger", &body)
	request.Header.Set("Content-Type", "application/json")

	Convey("Success create", t, func() {
		err := render.Bind(request, trigger)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Can't use warn_value with multiple targets")
	})

}

func TestExpressionModeMultipleTargetsErrorValue (t *testing.T){
	var body bytes.Buffer
	body.WriteString(`{
  "id": "GraphiteStoragesFreeSpace",
  "name": "Graphite storage free space low",
  "desc": "Graphite ClickHouse",
  "targets": [
    "aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)"
  ],
  "error_value": 5,
  "trigger_type": "expression",
  "tags": [
    "Normal",
    "DevOps",
    "DevOpsGraphite-duty"
  ],
  "ttl_state": "NODATA",
  "ttl": 600,
  "expression": "(t1 < 10 && t2 < 10) ? WARN:OK",
  "is_remote": false,
  "mute_new_metrics": false,
  "throttling": 0
	}`)

	trigger := &dto.Trigger{}

	request, _ := http.NewRequest("PUT", "/api/trigger", &body)
	request.Header.Set("Content-Type", "application/json")

	Convey("Success create", t, func() {
		err := render.Bind(request, trigger)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Can't use error_value with multiple targets")
	})
}

func TestSimpleModeOneTarget (t *testing.T){
	var body bytes.Buffer
	body.WriteString(`{
  "id": "GraphiteStoragesFreeSpace",
  "name": "Graphite storage free space low",
  "desc": "Graphite ClickHouse",
  "targets": [
    "aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)"
  ],
  "warn_value": 10,
  "error_value": 5,
  "trigger_type": "falling",
  "tags": [
    "Normal",
    "DevOps",
    "DevOpsGraphite-duty"
  ],
  "ttl_state": "NODATA",
  "ttl": 600,
  "is_remote": false,
  "mute_new_metrics": false,
  "throttling": 0
	}`)

	trigger := &dto.Trigger{}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	request, _ := http.NewRequest("PUT", "/api/trigger", &body)
	request.Header.Set("Content-Type", "application/json")

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)

	ctx := request.Context()
	ctx = context.WithValue(ctx, middleware.ContextKey("metricSourceProvider"), sourceProvider )
	request = request.WithContext(ctx)

	localSource.EXPECT().IsConfigured().Return(true, nil)
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil)
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil)
	fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData("", []float64{}, 0, 0)})

	Convey("Success create", t, func() {
		err := render.Bind(request, trigger)
		So(err, ShouldBeNil)
		So(trigger.ID, ShouldResemble, "GraphiteStoragesFreeSpace")
		So(len(trigger.Targets), ShouldEqual, 1)
		So(*trigger.ErrorValue, ShouldEqual, 5)
		So(*trigger.WarnValue, ShouldEqual, 10)
	})
}

func TestExpressionModeMultipleTargets (t *testing.T){
	var body bytes.Buffer
	body.WriteString(`{
  "id": "GraphiteStoragesFreeSpace",
  "name": "Graphite storage free space low",
  "desc": "Graphite ClickHouse",
  "targets": [
        "aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
        "aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
        "aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
        "aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)"
  ],
  "trigger_type": "expression",
  "tags": [
    "Normal",
    "DevOps",
    "DevOpsGraphite-duty"
  ],
  "ttl_state": "NODATA",
  "ttl": 600,
  "expression": "(t1 < 10 && t2 < 10) ? WARN:OK",
  "is_remote": false,
  "mute_new_metrics": false,
  "throttling": 0
	}`)

	trigger := &dto.Trigger{}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	request, _ := http.NewRequest("PUT", "/api/trigger", &body)
	request.Header.Set("Content-Type", "application/json")

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)

	ctx := request.Context()
	ctx = context.WithValue(ctx, middleware.ContextKey("metricSourceProvider"), sourceProvider )
	request = request.WithContext(ctx)

	localSource.EXPECT().IsConfigured().Return(true, nil)
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData("", []float64{}, 0, 0)})

	Convey("Success create", t, func() {
		err := render.Bind(request, trigger)
		So(err, ShouldBeNil)
		So(trigger.ID, ShouldResemble, "GraphiteStoragesFreeSpace")
		So(len(trigger.Targets), ShouldEqual, 4)
		So(trigger.ErrorValue, ShouldBeNil)
		So(trigger.WarnValue, ShouldBeNil)
	})
}

func TestSimpleModeMultipleTargets (t *testing.T){
	var body bytes.Buffer
	body.WriteString(`{
  "id": "GraphiteStoragesFreeSpace",
  "name": "Graphite storage free space low",
  "desc": "Graphite ClickHouse",
  "targets": [
        "aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
        "aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
        "aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
        "aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)"
  ],
  "trigger_type": "falling",
  "tags": [
    "Normal",
    "DevOps",
    "DevOpsGraphite-duty"
  ],
  "ttl_state": "NODATA",
  "ttl": 600,
  "expression": "(t1 < 10 && t2 < 10) ? WARN:OK",
  "is_remote": false,
  "mute_new_metrics": false,
  "throttling": 0
	}`)

	trigger := &dto.Trigger{}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	request, _ := http.NewRequest("PUT", "/api/trigger", &body)
	request.Header.Set("Content-Type", "application/json")

	Convey("Success create", t, func() {
		err := render.Bind(request, trigger)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "Can't use trigger_type not 'expression' for with multiple targets")
		So(trigger.ID, ShouldResemble, "GraphiteStoragesFreeSpace")
		So(len(trigger.Targets), ShouldEqual, 4)
		So(trigger.ErrorValue, ShouldBeNil)
		So(trigger.WarnValue, ShouldBeNil)
	})
}
