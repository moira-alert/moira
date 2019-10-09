package controller

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/remote"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

func TestDeleteTriggerMetric(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	trigger := moira.Trigger{ID: triggerID}
	lastCheck := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {},
		},
	}
	emptyLastCheck := moira.CheckData{
		Metrics: make(map[string]moira.MetricState),
	}

	Convey("Success delete from last check", t, func() {
		expectedLastCheck := lastCheck
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(expectedLastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, &expectedLastCheck, trigger.IsRemote)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, emptyLastCheck)
	})

	Convey("Success delete nothing to delete", t, func() {
		expectedLastCheck := emptyLastCheck
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(expectedLastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, &expectedLastCheck, trigger.IsRemote)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, emptyLastCheck)
	})

	Convey("No trigger", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, database.ErrNil)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger not found")))
	})

	Convey("No last check", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger check not found")))
	})

	Convey("Get trigger error", t, func() {
		expected := fmt.Errorf("get trigger error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, expected)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("AcquireTriggerCheckLock error", t, func() {
		expected := fmt.Errorf("acquire error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(expected)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("GetTriggerLastCheck error", t, func() {
		expected := fmt.Errorf("last check error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, expected)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("RemovePatternsMetrics error", t, func() {
		expected := fmt.Errorf("removePatternsMetrics err")
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(expected)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("SetTriggerLastCheck error", t, func() {
		expected := fmt.Errorf("removePatternsMetrics err")
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.IsRemote).Return(expected)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestDeleteTriggerNodataMetrics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	trigger := moira.Trigger{ID: triggerID}

	lastCheckWithManyStates := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {State: moira.StateNODATA},
			"super.metric2": {State: moira.StateNODATA},
			"super.metric3": {State: moira.StateNODATA},
			"super.metric4": {State: moira.StateOK},
			"super.metric5": {State: moira.StateERROR},
			"super.metric6": {State: moira.StateNODATA},
		},
		Score: 100,
	}

	lastCheckWithoutNodata := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric4": {State: moira.StateOK},
			"super.metric5": {State: moira.StateERROR},
		},
		Score: 100,
	}

	lastCheckSingleNodata := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {State: moira.StateNODATA},
		},
	}
	emptyLastCheck := moira.CheckData{
		Metrics: make(map[string]moira.MetricState),
	}

	lastCheckWithNodataOnly := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {State: moira.StateNODATA},
			"super.metric2": {State: moira.StateNODATA},
			"super.metric3": {State: moira.StateNODATA},
			"super.metric6": {State: moira.StateNODATA},
		},
	}

	Convey("Success delete from last check, one NODATA", t, func() {
		expectedLastCheck := lastCheckSingleNodata
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(expectedLastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, &expectedLastCheck, trigger.IsRemote)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, emptyLastCheck)
	})

	Convey("Success delete from last check, many NODATA", t, func() {
		expectedLastCheck := lastCheckWithNodataOnly
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(expectedLastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, &expectedLastCheck, trigger.IsRemote)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, emptyLastCheck)
	})

	Convey("Success delete from last check, many NODATA + other statuses", t, func() {
		expectedLastCheck := lastCheckWithManyStates
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(expectedLastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheckWithoutNodata, trigger.IsRemote)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, lastCheckWithoutNodata)
	})

	Convey("Success delete nothing to delete", t, func() {
		expectedLastCheck := emptyLastCheck
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(expectedLastCheck, nil)
		dataBase.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, &expectedLastCheck, trigger.IsRemote)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, emptyLastCheck)
	})

	Convey("No trigger", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, database.ErrNil)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger not found")))
	})

	Convey("No last check", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger check not found")))
	})

	Convey("Get trigger error", t, func() {
		expected := fmt.Errorf("get trigger error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, expected)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("AcquireTriggerCheckLock error", t, func() {
		expected := fmt.Errorf("acquire error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(expected)
		err := DeleteTriggerMetric(dataBase, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("GetTriggerLastCheck error", t, func() {
		expected := fmt.Errorf("last check error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, expected)
		err := DeleteTriggerNodataMetrics(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestGetTriggerMetrics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"

	var from int64 = 17
	var until int64 = 67
	var retention int64 = 10

	Convey("Trigger is remote but remote is not configured", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{ID: triggerID, Targets: []string{pattern}, IsRemote: true}, nil)
		remoteSource.EXPECT().IsConfigured().Return(false, nil)
		triggerMetrics, err := GetTriggerMetrics(dataBase, sourceProvider, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(metricSource.ErrMetricSourceIsNotConfigured))
		So(triggerMetrics, ShouldBeNil)
	})

	Convey("Trigger is remote but remote has bad config", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{ID: triggerID, Targets: []string{pattern}, IsRemote: true}, nil)
		remoteSource.EXPECT().IsConfigured().Return(false, remote.ErrRemoteStorageDisabled)
		triggerMetrics, err := GetTriggerMetrics(dataBase, sourceProvider, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(remote.ErrRemoteStorageDisabled))
		So(triggerMetrics, ShouldBeNil)
	})

	Convey("Has metrics", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{ID: triggerID, Targets: []string{pattern}}, nil)
		localSource.EXPECT().IsConfigured().Return(true, nil)
		localSource.EXPECT().Fetch(pattern, from, until, false).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)})
		triggerMetrics, err := GetTriggerMetrics(dataBase, sourceProvider, from, until, triggerID)
		So(err, ShouldBeNil)
		So(*triggerMetrics, ShouldResemble, dto.TriggerMetrics{"t1": map[string][]moira.MetricValue{metric: {{Value: 0, Timestamp: 17}, {Value: 1, Timestamp: 27}, {Value: 2, Timestamp: 37}, {Value: 3, Timestamp: 47}, {Value: 4, Timestamp: 57}}}})
	})

	Convey("GetTrigger error", t, func() {
		expected := fmt.Errorf("get trigger error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, expected)
		triggerMetrics, err := GetTriggerMetrics(dataBase, sourceProvider, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(triggerMetrics, ShouldBeNil)
	})

	Convey("No trigger", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, database.ErrNil)
		triggerMetrics, err := GetTriggerMetrics(dataBase, sourceProvider, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger not found")))
		So(triggerMetrics, ShouldBeNil)
	})

	Convey("Fetch error", t, func() {
		expectedError := remote.ErrRemoteTriggerResponse{InternalError: fmt.Errorf("some error"), Target: pattern}
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{ID: triggerID, Targets: []string{pattern}, IsRemote: true}, nil)
		remoteSource.EXPECT().IsConfigured().Return(true, nil)
		remoteSource.EXPECT().Fetch(pattern, from, until, false).Return(nil, expectedError)
		triggerMetrics, err := GetTriggerMetrics(dataBase, sourceProvider, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		So(triggerMetrics, ShouldBeNil)
	})
}
