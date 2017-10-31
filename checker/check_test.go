package checker

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/target"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"math"
	"testing"
)

func TestGetTimeSeriesState(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	var warnValue float64 = 10
	var errValue float64 = 20
	triggerChecker := TriggerChecker{
		Logger:  logger,
		Metrics: metrics.ConfigureCheckerMetrics("checker"),
		Until:   67,
		From:    17,
		trigger: &moira.Trigger{
			WarnValue:  &warnValue,
			ErrorValue: &errValue,
		},
	}
	fetchResponse := pb.FetchResponse{
		Name:      "main.metric",
		StartTime: int32(triggerChecker.From),
		StopTime:  int32(triggerChecker.Until),
		StepTime:  int32(10),
		Values:    []float64{1, 2, 3, 4, math.NaN()},
		IsAbsent:  []bool{false, true, false, false, false},
	}
	addFetchResponse := pb.FetchResponse{
		Name:      "additional.metric",
		StartTime: int32(triggerChecker.From),
		StopTime:  int32(triggerChecker.Until),
		StepTime:  int32(10),
		Values:    []float64{math.NaN(), 4, 3, 2, 1},
		IsAbsent:  []bool{false, false, false, false, false},
	}
	addFetchResponse.Name = "additional.metric"
	tts := &triggerTimeSeries{
		Main: []*target.TimeSeries{{
			MetricData: expr.MetricData{FetchResponse: fetchResponse},
		}},
		Additional: []*target.TimeSeries{{
			MetricData: expr.MetricData{FetchResponse: addFetchResponse},
		}},
	}
	metricLastState := moira.MetricState{
		Maintenance: 11111,
		Suppressed:  true,
	}

	Convey("Checkpoint more than valueTimestamp", t, func() {
		metricState, err := triggerChecker.getTimeSeriesState(tts, tts.Main[0], metricLastState, 37, 47)
		So(err, ShouldBeNil)
		So(metricState, ShouldBeNil)
	})

	Convey("Checkpoint lover than valueTimestamp", t, func() {
		Convey("Has all value by eventTimestamp step", func() {
			metricState, err := triggerChecker.getTimeSeriesState(tts, tts.Main[0], metricLastState, 42, 27)
			So(err, ShouldBeNil)
			So(metricState, ShouldResemble, &moira.MetricState{
				State:          OK,
				Timestamp:      42,
				Value:          &fetchResponse.Values[2],
				Maintenance:    metricLastState.Maintenance,
				Suppressed:     metricLastState.Suppressed,
				EventTimestamp: 0,
			})
		})

		Convey("No value in main timeSeries by eventTimestamp step", func() {
			metricState, err := triggerChecker.getTimeSeriesState(tts, tts.Main[0], metricLastState, 66, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("IsAbsent in main timeSeries by eventTimestamp step", func() {
			metricState, err := triggerChecker.getTimeSeriesState(tts, tts.Main[0], metricLastState, 29, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("No value in additional timeSeries by eventTimestamp step", func() {
			metricState, err := triggerChecker.getTimeSeriesState(tts, tts.Main[0], metricLastState, 26, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		metricState, err := triggerChecker.getTimeSeriesState(tts, tts.Main[0], metricLastState, 42, 27)
		So(err.Error(), ShouldResemble, "Invalid expression: Error value and Warning value can not be empty")
		So(metricState, ShouldBeNil)
	})
}

func TestGetTimeSeriesStepsStates(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	logging.SetLevel(logging.INFO, "Test")
	var warnValue float64 = 10
	var errValue float64 = 20
	triggerChecker := TriggerChecker{
		Logger: logger,
		Until:  67,
		From:   17,
		trigger: &moira.Trigger{
			WarnValue:  &warnValue,
			ErrorValue: &errValue,
		},
	}
	fetchResponse1 := pb.FetchResponse{
		Name:      "main.metric",
		StartTime: int32(triggerChecker.From),
		StopTime:  int32(triggerChecker.Until),
		StepTime:  int32(10),
		Values:    []float64{1, 2, 3, 4, math.NaN()},
		IsAbsent:  []bool{false, true, false, false, false},
	}
	fetchResponse2 := pb.FetchResponse{
		Name:      "main.metric",
		StartTime: int32(triggerChecker.From),
		StopTime:  int32(triggerChecker.Until),
		StepTime:  int32(10),
		Values:    []float64{1, 2, 3, 4, 5},
		IsAbsent:  []bool{false, false, false, false, false},
	}
	addFetchResponse := pb.FetchResponse{
		Name:      "additional.metric",
		StartTime: int32(triggerChecker.From),
		StopTime:  int32(triggerChecker.Until),
		StepTime:  int32(10),
		Values:    []float64{5, 4, 3, 2, 1},
		IsAbsent:  []bool{false, false, false, false, false},
	}
	addFetchResponse.Name = "additional.metric"
	tts := &triggerTimeSeries{
		Main:       []*target.TimeSeries{{MetricData: expr.MetricData{FetchResponse: fetchResponse1}}, {MetricData: expr.MetricData{FetchResponse: fetchResponse2}}},
		Additional: []*target.TimeSeries{{MetricData: expr.MetricData{FetchResponse: addFetchResponse}}},
	}
	metricLastState := moira.MetricState{
		Maintenance:    11111,
		Suppressed:     true,
		EventTimestamp: 11,
	}

	metricsState1 := moira.MetricState{
		State:          OK,
		Timestamp:      17,
		Value:          &fetchResponse2.Values[0],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState2 := moira.MetricState{
		State:          OK,
		Timestamp:      27,
		Value:          &fetchResponse2.Values[1],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState3 := moira.MetricState{
		State:          OK,
		Timestamp:      37,
		Value:          &fetchResponse2.Values[2],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState4 := moira.MetricState{
		State:          OK,
		Timestamp:      47,
		Value:          &fetchResponse2.Values[3],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState5 := moira.MetricState{
		State:          OK,
		Timestamp:      57,
		Value:          &fetchResponse2.Values[4],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	Convey("ValueTimestamp covers all TimeSeries range", t, func() {
		metricLastState.EventTimestamp = 11
		Convey("TimeSeries has all valid values", func() {
			metricStates, err := triggerChecker.getTimeSeriesStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("TimeSeries has invalid values", func() {
			metricStates, err := triggerChecker.getTimeSeriesStepsStates(tts, tts.Main[0], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState3, metricsState4})
		})

		Convey("Until + stepTime covers last value", func() {
			triggerChecker.Until = 56
			metricStates, err := triggerChecker.getTimeSeriesStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})
	})

	triggerChecker.Until = 67

	Convey("ValueTimestamp don't covers begin of TimeSeries", t, func() {
		Convey("Exclude 1 first element", func() {
			metricLastState.EventTimestamp = 22
			metricStates, err := triggerChecker.getTimeSeriesStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude 2 first elements", func() {
			metricLastState.EventTimestamp = 27
			metricStates, err := triggerChecker.getTimeSeriesStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude last element", func() {
			metricLastState.EventTimestamp = 11
			triggerChecker.Until = 47
			metricStates, err := triggerChecker.getTimeSeriesStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4})
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		metricLastState.EventTimestamp = 11
		triggerChecker.Until = 47
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		metricState, err := triggerChecker.getTimeSeriesStepsStates(tts, tts.Main[1], metricLastState)
		So(err.Error(), ShouldResemble, "Invalid expression: Error value and Warning value can not be empty")
		So(metricState, ShouldBeNil)
	})
}

func TestCheckForNODATA(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	logging.SetLevel(logging.INFO, "Test")
	metricLastState := moira.MetricState{
		EventTimestamp: 11,
		Maintenance:    11111,
		Suppressed:     true,
	}
	fetchResponse1 := pb.FetchResponse{
		Name: "main.metric",
	}
	timeSeries := &target.TimeSeries{
		MetricData: expr.MetricData{FetchResponse: fetchResponse1},
	}
	Convey("No TTL", t, func() {
		triggerChecker := TriggerChecker{}
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
		So(needToDeleteMetric, ShouldBeFalse)
		So(currentState, ShouldBeNil)
	})

	var ttl int64 = 600
	triggerChecker := TriggerChecker{
		Metrics: metrics.ConfigureCheckerMetrics("checker"),
		Logger:  logger,
		ttl:     ttl,
		lastCheck: &moira.CheckData{
			Timestamp: 1000,
		},
	}

	Convey("Last check is resent", t, func() {
		Convey("1", func() {
			metricLastState.Timestamp = 1100
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})
		Convey("2", func() {
			metricLastState.Timestamp = 401
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})
	})

	metricLastState.Timestamp = 399
	triggerChecker.ttlState = DEL

	Convey("TTLState is DEL and has EventTimeStamp", t, func() {
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
		So(needToDeleteMetric, ShouldBeTrue)
		So(currentState, ShouldBeNil)
	})

	Convey("Has new metricState", t, func() {
		Convey("TTLState is DEL, but no EventTimestamp", func() {
			metricLastState.EventTimestamp = 0
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       NODATA,
				Timestamp:   triggerChecker.lastCheck.Timestamp - triggerChecker.ttl,
				Value:       nil,
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})

		Convey("TTLState is OK and no EventTimestamp", func() {
			metricLastState.EventTimestamp = 0
			triggerChecker.ttlState = OK
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       triggerChecker.ttlState,
				Timestamp:   triggerChecker.lastCheck.Timestamp - triggerChecker.ttl,
				Value:       nil,
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})

		Convey("TTLState is OK and has EventTimestamp", func() {
			metricLastState.EventTimestamp = 111
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       triggerChecker.ttlState,
				Timestamp:   triggerChecker.lastCheck.Timestamp - triggerChecker.ttl,
				Value:       nil,
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})
	})
}

func TestHasMetrics(t *testing.T) {
	var ttl int64 = 100
	triggerCheckerWithoutTTL := &TriggerChecker{}
	triggerCheckerWithTTL := &TriggerChecker{
		ttl:      ttl,
		ttlState: NODATA,
		lastCheck: &moira.CheckData{
			Metrics: make(map[string]moira.MetricState),
		},
	}
	tts := &triggerTimeSeries{
		Main:       []*target.TimeSeries{{MetricData: expr.MetricData{}}, {MetricData: expr.MetricData{}}},
		Additional: []*target.TimeSeries{{MetricData: expr.MetricData{}}},
	}

	Convey("TriggerTimeSeries has metrics", t, func() {
		Convey("Trigger checker no ttl", func() {
			hasMetrics, sendEvent := triggerCheckerWithoutTTL.checkForNoMetrics(tts)
			So(hasMetrics, ShouldBeTrue)
			So(sendEvent, ShouldBeFalse)
		})

		Convey("Trigger checker has ttl", func() {
			hasMetrics, sendEvent := triggerCheckerWithTTL.checkForNoMetrics(tts)
			So(hasMetrics, ShouldBeTrue)
			So(sendEvent, ShouldBeFalse)
		})
	})

	tts = &triggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}

	Convey("TriggerTimeSeries no metrics", t, func() {
		Convey("Trigger checker no ttl", func() {
			hasMetrics, sendEvent := triggerCheckerWithoutTTL.checkForNoMetrics(tts)
			So(hasMetrics, ShouldBeFalse)
			So(sendEvent, ShouldBeFalse)
		})

		Convey("Trigger checker has ttl", func() {
			Convey("LastCheck no metrics data", func() {
				hasMetrics, sendEvent := triggerCheckerWithTTL.checkForNoMetrics(tts)
				So(hasMetrics, ShouldBeFalse)
				So(sendEvent, ShouldBeFalse)
			})

			Convey("LastCheck has metrics data", func() {
				triggerCheckerWithTTL.lastCheck.Metrics["123"] = moira.MetricState{}
				hasMetrics, sendEvent := triggerCheckerWithTTL.checkForNoMetrics(tts)
				So(hasMetrics, ShouldBeFalse)
				So(sendEvent, ShouldBeTrue)
			})
		})
	})
}

func TestCheckErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	defer mockCtrl.Finish()

	var retention int64 = 10
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"
	metricErr := fmt.Errorf("Ooops, metric error")

	var ttl int64 = 30

	triggerChecker := TriggerChecker{
		TriggerID: "SuperId",
		Database:  dataBase,
		Logger:    logger,
		Config: &Config{
			MetricsTTL: 10,
		},
		Metrics:  metrics.ConfigureCheckerMetrics("checker"),
		From:     17,
		Until:    67,
		ttl:      ttl,
		ttlState: NODATA,
		trigger: &moira.Trigger{
			Targets:  []string{pattern},
			Patterns: []string{pattern},
		},
		lastCheck: &moira.CheckData{
			State:     EXCEPTION,
			Timestamp: 57,
			Metrics: map[string]moira.MetricState{
				metric: {
					State:     OK,
					Timestamp: 26,
				},
			},
		},
	}

	Convey("GetTimeSeries error", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, triggerChecker.From, triggerChecker.Until).Return(nil, metricErr)
		dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.TriggerID, &moira.CheckData{
			Metrics:        triggerChecker.lastCheck.Metrics,
			State:          EXCEPTION,
			Timestamp:      triggerChecker.Until,
			EventTimestamp: 0,
			Score:          100000,
			Message:        "Trigger evaluation exception",
		}).Return(nil)
		err := triggerChecker.Check()
		So(err, ShouldBeNil)
	})
}

func TestHandleTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	logging.SetLevel(logging.INFO, "Test")
	defer mockCtrl.Finish()

	var retention int64 = 10
	var warnValue float64 = 10
	var errValue float64 = 20
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"
	var ttl int64 = 600
	lastCheck := moira.CheckData{
		Metrics:   make(map[string]moira.MetricState),
		State:     NODATA,
		Timestamp: 66,
	}
	metricValues := []*moira.MetricValue{
		{
			RetentionTimestamp: 3620,
			Timestamp:          3623,
			Value:              0,
		},
		{
			RetentionTimestamp: 3630,
			Timestamp:          3633,
			Value:              1,
		},
		{
			RetentionTimestamp: 3640,
			Timestamp:          3643,
			Value:              2,
		},
		{
			RetentionTimestamp: 3650,
			Timestamp:          3653,
			Value:              3,
		},
		{
			RetentionTimestamp: 3660,
			Timestamp:          3663,
			Value:              4,
		},
	}
	dataList := map[string][]*moira.MetricValue{
		metric: metricValues,
	}
	triggerChecker := TriggerChecker{
		TriggerID: "SuperId",
		Database:  dataBase,
		Logger:    logger,
		Config: &Config{
			MetricsTTL: 3600,
		},
		From:     3617,
		Until:    3667,
		ttl:      ttl,
		ttlState: NODATA,
		trigger: &moira.Trigger{
			ErrorValue: &errValue,
			WarnValue:  &warnValue,
			Targets:    []string{pattern},
			Patterns:   []string{pattern},
		},
		lastCheck: &lastCheck,
	}

	Convey("First Event", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, triggerChecker.From, triggerChecker.Until).Return(dataList, nil)
		var val float64
		var val1 float64 = 4
		dataBase.EXPECT().RemoveMetricValues(metric, triggerChecker.Until-triggerChecker.Config.MetricsTTL)
		dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
			TriggerID: triggerChecker.TriggerID,
			Timestamp: 3617,
			State:     OK,
			OldState:  NODATA,
			Metric:    metric,
			Value:     &val,
			Message:   nil}, true).Return(nil)
		checkData, err := triggerChecker.handleTrigger()
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      3657,
					EventTimestamp: 3617,
					State:          OK,
					Value:          &val1,
				},
			},
			Timestamp: triggerChecker.Until,
			State:     OK,
			Score:     0,
		})
		mockCtrl.Finish()
	})

	var val float64 = 3
	lastCheck = moira.CheckData{
		Metrics: map[string]moira.MetricState{
			metric: {
				Timestamp:      3647,
				EventTimestamp: 3607,
				State:          OK,
				Value:          &val,
			},
		},
		State:     OK,
		Timestamp: 3655,
	}

	Convey("Last check is not empty", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, triggerChecker.From, triggerChecker.Until).Return(dataList, nil)
		dataBase.EXPECT().RemoveMetricValues(metric, triggerChecker.Until-triggerChecker.Config.MetricsTTL)
		checkData, err := triggerChecker.handleTrigger()
		So(err, ShouldBeNil)
		var val1 float64 = 4
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      3657,
					EventTimestamp: 3607,
					State:          OK,
					Value:          &val1,
				},
			},
			Timestamp: triggerChecker.Until,
			State:     OK,
			Score:     0,
		})
		mockCtrl.Finish()
	})

	Convey("No data too long", t, func() {
		triggerChecker.From = 4217
		triggerChecker.Until = 4267
		lastCheck.Timestamp = 4267
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, triggerChecker.From, triggerChecker.Until).Return(dataList, nil)
		dataBase.EXPECT().RemoveMetricValues(metric, triggerChecker.Until-triggerChecker.Config.MetricsTTL)
		dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
			TriggerID: triggerChecker.TriggerID,
			Timestamp: lastCheck.Timestamp - triggerChecker.ttl,
			State:     NODATA,
			OldState:  OK,
			Metric:    metric,
			Value:     nil,
			Message:   nil}, true).Return(nil)
		checkData, err := triggerChecker.handleTrigger()
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      lastCheck.Timestamp - triggerChecker.ttl,
					EventTimestamp: lastCheck.Timestamp - triggerChecker.ttl,
					State:          NODATA,
					Value:          nil,
				},
			},
			Timestamp: triggerChecker.Until,
			State:     OK,
			Score:     0,
		})

		mockCtrl.Finish()
	})

	Convey("No data too long and ttlState is delete", t, func() {
		triggerChecker.From = 4217
		triggerChecker.Until = 4267
		triggerChecker.ttlState = DEL
		lastCheck.Timestamp = 4267
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, triggerChecker.From, triggerChecker.Until).Return(dataList, nil)
		dataBase.EXPECT().RemoveMetricValues(metric, triggerChecker.Until-triggerChecker.Config.MetricsTTL)
		dataBase.EXPECT().RemovePatternsMetrics(triggerChecker.trigger.Patterns).Return(nil)
		checkData, err := triggerChecker.handleTrigger()
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			Timestamp: triggerChecker.Until,
			State:     OK,
			Score:     0,
		})
		mockCtrl.Finish()
	})
}
