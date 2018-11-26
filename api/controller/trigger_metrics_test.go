package controller

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/remote"
)

func TestDeleteTriggerMetric(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()
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
	triggerID := uuid.NewV4().String()
	trigger := moira.Trigger{ID: triggerID}

	lastCheckWithManyStates := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {State: "NODATA"},
			"super.metric2": {State: "NODATA"},
			"super.metric3": {State: "NODATA"},
			"super.metric4": {State: "OK"},
			"super.metric5": {State: "ERROR"},
			"super.metric6": {State: "NODATA"},
		},
		Score: 100,
	}

	lastCheckWithoutNodata := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric4": {State: "OK"},
			"super.metric5": {State: "ERROR"},
		},
		Score: 100,
	}

	lastCheckSingleNodata := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {State: "NODATA"},
		},
	}
	emptyLastCheck := moira.CheckData{
		Metrics: make(map[string]moira.MetricState),
	}

	lastCheckWithNodataOnly := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {State: "NODATA"},
			"super.metric2": {State: "NODATA"},
			"super.metric3": {State: "NODATA"},
			"super.metric6": {State: "NODATA"},
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
	triggerID := uuid.NewV4().String()
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"
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
	var remoteCfg *remote.Config

	Convey("Has metrics", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{ID: triggerID, Targets: []string{pattern}}, nil)
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
		triggerMetrics, err := GetTriggerMetrics(dataBase, remoteCfg, from, until, triggerID)
		So(err, ShouldBeNil)
		So(*triggerMetrics, ShouldResemble, dto.TriggerMetrics{Main: map[string][]*moira.MetricValue{metric: {{Value: 0, Timestamp: 17}, {Value: 1, Timestamp: 27}, {Value: 2, Timestamp: 37}, {Value: 3, Timestamp: 47}, {Value: 4, Timestamp: 57}}}, Additional: make(map[string][]*moira.MetricValue)})	})

	Convey("GetTrigger error", t, func() {
		expected := fmt.Errorf("get trigger error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, expected)
		triggerMetrics, err := GetTriggerMetrics(dataBase, remoteCfg, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(triggerMetrics, ShouldBeNil)
	})

	Convey("No trigger", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, database.ErrNil)
		triggerMetrics, err := GetTriggerMetrics(dataBase, remoteCfg, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger not found")))
		So(triggerMetrics, ShouldBeNil)
	})

	Convey("GetMetricsValues error", t, func() {
		expected := fmt.Errorf("getMetricsValues error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{ID: triggerID, Targets: []string{pattern}}, nil)
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(nil, expected)
		triggerMetrics, err := GetTriggerMetrics(dataBase, remoteCfg, from, until, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(triggerMetrics, ShouldBeNil)
	})

}
