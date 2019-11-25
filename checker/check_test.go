package checker

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/metrics"
)

func TestGetMetricDataState(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	var warnValue float64 = 10
	var errValue float64 = 20
	checkerMetrics := metrics.ConfigureCheckerMetrics(metrics.NewDummyRegistry(), false)
	triggerChecker := TriggerChecker{
		logger:  logger,
		metrics: checkerMetrics.LocalMetrics,
		until:   67,
		from:    17,
		trigger: &moira.Trigger{
			WarnValue:   &warnValue,
			ErrorValue:  &errValue,
			TriggerType: moira.RisingTrigger,
		},
	}
	metricData := metricSource.MetricData{
		Name:      "main.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{1, math.NaN(), 3, 4, math.NaN()},
	}
	addMetricData := metricSource.MetricData{
		Name:      "additional.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{math.NaN(), 4, 3, 2, 1},
	}
	addMetricData.Name = "additional.metric"
	tts := metricSource.MakeTriggerMetricsData(
		[]*metricSource.MetricData{&metricData},
		[]*metricSource.MetricData{&addMetricData},
	)
	metricLastState := moira.MetricState{
		Maintenance: 11111,
		Suppressed:  true,
	}

	Convey("Checkpoint more than valueTimestamp", t, func() {
		metricState, err := triggerChecker.getMetricDataState(tts, tts.Main[0], metricLastState, 37, 47)
		So(err, ShouldBeNil)
		So(metricState, ShouldBeNil)
	})

	Convey("Checkpoint lover than valueTimestamp", t, func() {
		Convey("Has all value by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(tts, tts.Main[0], metricLastState, 42, 27)
			So(err, ShouldBeNil)
			So(metricState, ShouldResemble, &moira.MetricState{
				State:          moira.StateOK,
				Timestamp:      42,
				Value:          &metricData.Values[2],
				Maintenance:    metricLastState.Maintenance,
				Suppressed:     metricLastState.Suppressed,
				EventTimestamp: 0,
			})
		})

		Convey("No value in main metric data by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(tts, tts.Main[0], metricLastState, 66, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("IsAbsent in main metric data by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(tts, tts.Main[0], metricLastState, 29, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("No value in additional metric data by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(tts, tts.Main[0], metricLastState, 26, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		metricState, err := triggerChecker.getMetricDataState(tts, tts.Main[0], metricLastState, 42, 27)
		So(err.Error(), ShouldResemble, "error value and warning value can not be empty")
		So(metricState, ShouldBeNil)
	})
}

func TestGetMetricsDataToCheck(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	Convey("Get metrics data to check:", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "ID",
			logger:    logger,
			from:      0,
			until:     60,
			lastCheck: &moira.CheckData{},
		}
		Convey("last check has no metrics", func() {
			Convey("fetched metrics is empty", func() {
				actual, err := triggerChecker.getMetricsToCheck([]*metricSource.MetricData{})
				So(actual, ShouldHaveLength, 0)
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has metrics", func() {
				actual, err := triggerChecker.getMetricsToCheck([]*metricSource.MetricData{metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0)})
				So(actual, ShouldHaveLength, 1)
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has duplicate metrics", func() {
				actual, err := triggerChecker.getMetricsToCheck(
					[]*metricSource.MetricData{
						metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0),
						metricSource.MakeMetricData("123", []float64{4, 5, 6}, 10, 0),
					})
				So(actual, ShouldResemble, []*metricSource.MetricData{metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0)})
				So(err, ShouldResemble, ErrTriggerHasSameMetricNames{names: []string{"123"}})
			})
		})

		Convey("last check has metrics", func() {
			triggerChecker.lastCheck = &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"first":  {},
					"second": {},
					"third":  {},
				}}

			Convey("fetched metrics is empty", func() {
				actual, err := triggerChecker.getMetricsToCheck([]*metricSource.MetricData{})
				So(actual, ShouldHaveLength, 3)
				for _, actualMetricData := range actual {
					So(actualMetricData.Values, ShouldHaveLength, 1)
					So(actualMetricData.StepTime, ShouldResemble, int64(60))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has only wildcards, step is 0", func() {
				actual, err := triggerChecker.getMetricsToCheck([]*metricSource.MetricData{{Name: "wildcard", Wildcard: true}})
				So(actual, ShouldHaveLength, 3)
				for _, actualMetricData := range actual {
					So(actualMetricData.Values, ShouldHaveLength, 1)
					So(actualMetricData.StepTime, ShouldResemble, int64(60))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has only wildcards, step is 10", func() {
				actual, err := triggerChecker.getMetricsToCheck([]*metricSource.MetricData{{Name: "wildcard", Wildcard: true, StepTime: 10}})
				So(actual, ShouldHaveLength, 3)
				for _, actualMetricData := range actual {
					So(actualMetricData.Values, ShouldHaveLength, 6)
					So(actualMetricData.StepTime, ShouldResemble, int64(10))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has one of last check metrics", func() {
				actual, err := triggerChecker.getMetricsToCheck([]*metricSource.MetricData{
					metricSource.MakeMetricData("first", []float64{1, 2, 3, 4, 5, 6}, 10, 0),
				})
				So(actual, ShouldHaveLength, 3)
				for _, actualMetricData := range actual {
					So(actualMetricData.Values, ShouldHaveLength, 6)
					So(actualMetricData.StepTime, ShouldResemble, int64(10))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has one of last check metrics and one new", func() {
				actual, err := triggerChecker.getMetricsToCheck([]*metricSource.MetricData{
					metricSource.MakeMetricData("first", []float64{1, 2, 3, 4, 5, 6}, 10, 0),
					metricSource.MakeMetricData("fourth", []float64{7, 8, 9, 1, 2, 3}, 10, 0),
				})
				So(actual, ShouldHaveLength, 4)
				for _, actualMetricData := range actual {
					So(actualMetricData.Values, ShouldHaveLength, 6)
					So(actualMetricData.StepTime, ShouldResemble, int64(10))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestGetMetricStepsStates(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	logging.SetLevel(logging.INFO, "Test")
	var warnValue float64 = 10
	var errValue float64 = 20
	triggerChecker := TriggerChecker{
		logger: logger,
		until:  67,
		from:   17,
		trigger: &moira.Trigger{
			WarnValue:   &warnValue,
			ErrorValue:  &errValue,
			TriggerType: moira.RisingTrigger,
		},
	}
	metricData1 := &metricSource.MetricData{
		Name:      "main.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{1, math.NaN(), 3, 4, math.NaN()},
	}
	metricData2 := &metricSource.MetricData{
		Name:      "main.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{1, 2, 3, 4, 5},
	}
	addMetricData := &metricSource.MetricData{
		Name:      "additional.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{5, 4, 3, 2, 1},
	}
	addMetricData.Name = "additional.metric"
	tts := &metricSource.TriggerMetricsData{
		Main:       []*metricSource.MetricData{metricData1, metricData2},
		Additional: []*metricSource.MetricData{addMetricData},
	}
	metricLastState := moira.MetricState{
		Maintenance:    11111,
		Suppressed:     true,
		EventTimestamp: 11,
	}

	metricsState1 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      17,
		Value:          &metricData2.Values[0],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState2 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      27,
		Value:          &metricData2.Values[1],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState3 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      37,
		Value:          &metricData2.Values[2],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState4 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      47,
		Value:          &metricData2.Values[3],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	metricsState5 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      57,
		Value:          &metricData2.Values[4],
		Maintenance:    metricLastState.Maintenance,
		Suppressed:     metricLastState.Suppressed,
		EventTimestamp: 0,
	}

	Convey("ValueTimestamp covers all metric range", t, func() {
		metricLastState.EventTimestamp = 11
		Convey("Metric has all valid values", func() {
			metricStates, err := triggerChecker.getMetricStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("Metric has invalid values", func() {
			metricStates, err := triggerChecker.getMetricStepsStates(tts, tts.Main[0], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState3, metricsState4})
		})

		Convey("Until + stepTime covers last value", func() {
			triggerChecker.until = 56
			metricStates, err := triggerChecker.getMetricStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})
	})

	triggerChecker.until = 67

	Convey("ValueTimestamp don't covers begin of metric data", t, func() {
		Convey("Exclude 1 first element", func() {
			metricLastState.EventTimestamp = 22
			metricStates, err := triggerChecker.getMetricStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude 2 first elements", func() {
			metricLastState.EventTimestamp = 27
			metricStates, err := triggerChecker.getMetricStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude last element", func() {
			metricLastState.EventTimestamp = 11
			triggerChecker.until = 47
			metricStates, err := triggerChecker.getMetricStepsStates(tts, tts.Main[1], metricLastState)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4})
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		metricLastState.EventTimestamp = 11
		triggerChecker.until = 47
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		metricState, err := triggerChecker.getMetricStepsStates(tts, tts.Main[1], metricLastState)
		So(err.Error(), ShouldResemble, "error value and warning value can not be empty")
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
	metricData1 := &metricSource.MetricData{
		Name: "main.metric",
	}
	Convey("No TTL", t, func() {
		triggerChecker := TriggerChecker{}
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricData1, metricLastState)
		So(needToDeleteMetric, ShouldBeFalse)
		So(currentState, ShouldBeNil)
	})

	var ttl int64 = 600

	checkerMetrics := metrics.ConfigureCheckerMetrics(metrics.NewDummyRegistry(), false)
	triggerChecker := TriggerChecker{
		metrics: checkerMetrics.LocalMetrics,
		logger:  logger,
		ttl:     ttl,
		lastCheck: &moira.CheckData{
			Timestamp: 1000,
		},
	}

	Convey("Last check is resent", t, func() {
		Convey("1", func() {
			metricLastState.Timestamp = 1100
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricData1, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})
		Convey("2", func() {
			metricLastState.Timestamp = 401
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricData1, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})
	})

	metricLastState.Timestamp = 399
	triggerChecker.ttlState = moira.TTLStateDEL

	Convey("TTLState is DEL and has EventTimeStamp", t, func() {
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricData1, metricLastState)
		So(needToDeleteMetric, ShouldBeTrue)
		So(currentState, ShouldBeNil)
	})

	Convey("Has new metricState", t, func() {
		Convey("TTLState is DEL, but no EventTimestamp", func() {
			metricLastState.EventTimestamp = 0
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricData1, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       moira.StateNODATA,
				Timestamp:   triggerChecker.lastCheck.Timestamp,
				Value:       nil,
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})

		Convey("TTLState is OK and no EventTimestamp", func() {
			metricLastState.EventTimestamp = 0
			triggerChecker.ttlState = moira.TTLStateOK
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricData1, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       triggerChecker.ttlState.ToMetricState(),
				Timestamp:   triggerChecker.lastCheck.Timestamp,
				Value:       nil,
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})

		Convey("TTLState is OK and has EventTimestamp", func() {
			metricLastState.EventTimestamp = 111
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricData1, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       triggerChecker.ttlState.ToMetricState(),
				Timestamp:   triggerChecker.lastCheck.Timestamp,
				Value:       nil,
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})
	})
}

func TestCheck(t *testing.T) {
	Convey("Check Errors", t, func() {
		mockCtrl := gomock.NewController(t)
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		source := mock_metric_source.NewMockMetricSource(mockCtrl)
		fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
		logger, _ := logging.GetLogger("Test")
		defer mockCtrl.Finish()

		var retention int64 = 10
		var warnValue float64 = 10
		var errValue float64 = 20
		pattern := "super.puper.pattern"
		metric := "super.puper.metric"
		message := "ooops, metric error"
		metricErr := fmt.Errorf(message)
		messageException := `Unknown graphite function: "WrongFunction"`
		unknownFunctionExc := local.ErrorUnknownFunction(fmt.Errorf(messageException))

		var ttl int64 = 30

		checkerMetrics := metrics.ConfigureCheckerMetrics(metrics.NewDummyRegistry(), false)
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			source:    source,
			logger:    logger,
			config: &Config{
				MetricsTTLSeconds: 10,
			},
			metrics:  checkerMetrics.LocalMetrics,
			from:     17,
			until:    67,
			ttl:      ttl,
			ttlState: moira.TTLStateNODATA,
			trigger: &moira.Trigger{
				Name:        "Super trigger",
				ErrorValue:  &errValue,
				WarnValue:   &warnValue,
				TriggerType: moira.RisingTrigger,
				Targets:     []string{pattern},
				Patterns:    []string{pattern},
			},
			lastCheck: &moira.CheckData{
				State:     moira.StateOK,
				Timestamp: 57,
				Metrics: map[string]moira.MetricState{
					metric: {
						State:     moira.StateOK,
						Timestamp: 26,
					},
				},
			},
		}

		Convey("Fetch error", func() {
			lastCheck := moira.CheckData{
				Metrics:        triggerChecker.lastCheck.Metrics,
				State:          moira.StateOK,
				Timestamp:      triggerChecker.until,
				EventTimestamp: triggerChecker.until,
				Score:          0,
				Message:        "",
			}

			gomock.InOrder(
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(nil, metricErr),
				dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil),
			)
			err := triggerChecker.Check()
			So(err, ShouldBeNil)
		})

		Convey("Switch trigger to EXCEPTION and back", func() {
			Convey("Switch state to EXCEPTION. Event should be created", func() {
				event := moira.NotificationEvent{
					IsTriggerEvent: true,
					TriggerID:      triggerChecker.triggerID,
					State:          moira.StateEXCEPTION,
					OldState:       moira.StateOK,
					Timestamp:      67,
					Metric:         triggerChecker.trigger.Name,
				}

				lastCheck := moira.CheckData{
					Metrics:                      triggerChecker.lastCheck.Metrics,
					State:                        moira.StateEXCEPTION,
					Timestamp:                    triggerChecker.until,
					EventTimestamp:               triggerChecker.until,
					Score:                        100000,
					Message:                      messageException,
					LastSuccessfulCheckTimestamp: 0,
				}

				gomock.InOrder(
					source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(nil, unknownFunctionExc),
					dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil),
					dataBase.EXPECT().GetMetricsTTLSeconds().Return(int64(10))
					dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil),
				)
				err := triggerChecker.Check()
				So(err, ShouldBeNil)
			})

			Convey("Switch state to OK. Event should be created", func() {
				triggerChecker.lastCheck.State = moira.StateEXCEPTION
				triggerChecker.lastCheck.EventTimestamp = 67
				triggerChecker.lastCheck.LastSuccessfulCheckTimestamp = triggerChecker.until
				lastValue := float64(4)

				eventMetrics := map[string]moira.MetricState{
					metric: {
						EventTimestamp: 17,
						State:          moira.StateOK,
						Suppressed:     false,
						Timestamp:      57,
						Value:          &lastValue,
					},
				}

				event := moira.NotificationEvent{
					IsTriggerEvent: true,
					TriggerID:      triggerChecker.triggerID,
					State:          moira.StateOK,
					OldState:       moira.StateEXCEPTION,
					Timestamp:      67,
					Metric:         triggerChecker.trigger.Name,
				}

				lastCheck := moira.CheckData{
					Metrics:                      eventMetrics,
					State:                        moira.StateOK,
					Timestamp:                    triggerChecker.until,
					EventTimestamp:               triggerChecker.until,
					Score:                        0,
					LastSuccessfulCheckTimestamp: triggerChecker.until,
				}

				gomock.InOrder(
					source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil),
					fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}),
					fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil),
					dataBase.EXPECT().RemoveMetricsValues([]string{metric}, int64(57)).Return(nil),
					dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil),
					dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil),
				)
				err := triggerChecker.Check()
				So(err, ShouldBeNil)
			})
		})

		Convey("Trigger switch to Error", func() {
			value := float64(25)
			lastCheck := moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						EventTimestamp:  57,
						State:           moira.StateERROR,
						Timestamp:       57,
						MaintenanceInfo: moira.MaintenanceInfo{},
						Value:           &value,
					},
				},
				Score:                        100,
				State:                        moira.StateOK,
				Timestamp:                    triggerChecker.until,
				EventTimestamp:               0,
				LastSuccessfulCheckTimestamp: triggerChecker.until,
			}
			event := moira.NotificationEvent{
				IsTriggerEvent: false,
				TriggerID:      triggerChecker.triggerID,
				State:          moira.StateERROR,
				OldState:       moira.StateOK,
				Timestamp:      57,
				Metric:         metric,
				Value:          &value,
			}

			dataBase.EXPECT().RemoveMetricsValues([]string{metric}, int64(57)).Return(nil)
			source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{
				metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 25}, retention, triggerChecker.from),
			})
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
			dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil)
			//dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil)
			err := triggerChecker.Check()
			So(err, ShouldBeNil)
		})
		Convey("Duplicate error", func() {
			value := float64(4)
			lastCheck := moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						EventTimestamp:  17,
						State:           moira.StateOK,
						Timestamp:       57,
						MaintenanceInfo: moira.MaintenanceInfo{},
						Value:           &value,
					},
				},
				Score:          100,
				State:          moira.StateERROR,
				Timestamp:      triggerChecker.until,
				EventTimestamp: triggerChecker.until,
				Message:        "Several metrics have an identical name: super.puper.metric",
			}
			event := moira.NotificationEvent{
				IsTriggerEvent: true,
				TriggerID:      triggerChecker.triggerID,
				State:          moira.StateERROR,
				OldState:       moira.StateOK,
				Timestamp:      67,
				Metric:         triggerChecker.trigger.Name,
			}

			dataBase.EXPECT().RemoveMetricsValues([]string{metric}, int64(57)).Return(nil)
			source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{
				metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from),
				metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from),
			})
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
			dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil)
			err := triggerChecker.Check()
			So(err, ShouldBeNil)
		})
	})
}

func TestIgnoreNodataToOk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	source := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	logging.SetLevel(logging.INFO, "Test")
	defer mockCtrl.Finish()

	var retention int64 = 10
	var metricsTTL int64 = 3600
	var warnValue float64 = 10
	var errValue float64 = 20
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"
	var ttl int64 = 600
	lastCheck := moira.CheckData{
		Metrics:   make(map[string]moira.MetricState),
		State:     moira.StateNODATA,
		Timestamp: 66,
	}
	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
		source:    source,
		logger:    logger,
		config:    &Config{},
		from:      3617,
		until:     3667,
		ttl:       ttl,
		ttlState:  moira.TTLStateNODATA,
		trigger: &moira.Trigger{
			ErrorValue:  &errValue,
			WarnValue:   &warnValue,
			TriggerType: moira.RisingTrigger,
			Targets:     []string{pattern},
			Patterns:    []string{pattern},
		},
		lastCheck: &lastCheck,
	}

	Convey("First Event, NODATA - OK is ignored", t, func() {
		triggerChecker.trigger.MuteNewMetrics = true
		source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)})
		fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-3600)
		checkData, err := triggerChecker.checkTrigger()
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      time.Now().Unix(),
					EventTimestamp: time.Now().Unix(),
					State:          moira.StateOK,
					Value:          nil,
				},
			},
			Timestamp: triggerChecker.until,
			State:     moira.StateNODATA,
			Score:     0,
		})
	})
}

func TestHandleTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	source := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	logging.SetLevel(logging.INFO, "Test")
	defer mockCtrl.Finish()

	var retention int64 = 10
	var metricsTTL int64 = 3600
	var warnValue float64 = 10
	var errValue float64 = 20
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"
	var ttl int64 = 600
	lastCheck := moira.CheckData{
		Metrics:   make(map[string]moira.MetricState),
		State:     moira.StateNODATA,
		Timestamp: 66,
	}
	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
		source:    source,
		logger:    logger,
		config:    &Config{},
		from:      3617,
		until:     3667,
		ttl:       ttl,
		ttlState:  moira.TTLStateNODATA,
		trigger: &moira.Trigger{
			ErrorValue:  &errValue,
			WarnValue:   &warnValue,
			TriggerType: moira.RisingTrigger,
			Targets:     []string{pattern},
			Patterns:    []string{pattern},
		},
		lastCheck: &lastCheck,
	}

	Convey("First Event", t, func() {
		source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)})
		fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
		var val float64
		var val1 float64 = 4
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-3600)
		dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
			TriggerID: triggerChecker.triggerID,
			Timestamp: 3617,
			State:     moira.StateOK,
			OldState:  moira.StateNODATA,
			Metric:    metric,
			Value:     &val,
			Message:   nil}, true).Return(nil)
		checkData, err := triggerChecker.checkTrigger()
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      3657,
					EventTimestamp: 3617,
					State:          moira.StateOK,
					Value:          &val1,
				},
			},
			Timestamp: triggerChecker.until,
			State:     moira.StateNODATA,
			Score:     0,
		})
	})

	var val float64 = 3
	lastCheck = moira.CheckData{
		Metrics: map[string]moira.MetricState{
			metric: {
				Timestamp:      3647,
				EventTimestamp: 3607,
				State:          moira.StateOK,
				Value:          &val,
			},
		},
		State:     moira.StateOK,
		Timestamp: 3655,
	}

	Convey("Last check is not empty", t, func() {
		source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)})
		fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-3600)
		checkData, err := triggerChecker.checkTrigger()
		So(err, ShouldBeNil)
		var val1 float64 = 4
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      3657,
					EventTimestamp: 3607,
					State:          moira.StateOK,
					Value:          &val1,
				},
			},
			Timestamp: triggerChecker.until,
			State:     moira.StateOK,
			Score:     0,
		})
	})

	Convey("No data too long", t, func() {
		triggerChecker.from = 4217
		triggerChecker.until = 4267
		lastCheck.Timestamp = 4267
		source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)})
		fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-3600)
		dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
			TriggerID: triggerChecker.triggerID,
			Timestamp: lastCheck.Timestamp,
			State:     moira.StateNODATA,
			OldState:  moira.StateOK,
			Metric:    metric,
			Value:     nil,
			Message:   nil}, true).Return(nil)
		checkData, err := triggerChecker.checkTrigger()
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      lastCheck.Timestamp,
					EventTimestamp: lastCheck.Timestamp,
					State:          moira.StateNODATA,
					Value:          nil,
				},
			},
			Timestamp: triggerChecker.until,
			State:     moira.StateOK,
			Score:     0,
		})
	})

	Convey("Has duplicated metric names, should return trigger has same timeseries names error", t, func() {
		metric1 := "super.puper.metric"
		metric2 := "super.drupper.metric"
		pattern1 := "super.*.metric"
		f := 3.0

		triggerChecker1 := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			source:    source,
			logger:    logger,
			config:    &Config{},
			from:      3617,
			until:     3667,
			ttl:       ttl,
			ttlState:  moira.TTLStateNODATA,
			trigger: &moira.Trigger{
				ErrorValue:  &errValue,
				WarnValue:   &warnValue,
				TriggerType: moira.RisingTrigger,
				Targets:     []string{"aliasByNode(super.*.metric, 0)"},
				Patterns:    []string{pattern1},
			},
			lastCheck: &moira.CheckData{
				Metrics:   make(map[string]moira.MetricState),
				State:     moira.StateNODATA,
				Timestamp: 3647,
			},
		}

		source.EXPECT().Fetch(triggerChecker1.trigger.Targets[0], triggerChecker1.from, triggerChecker1.until, false).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{
			metricSource.MakeMetricData("super", []float64{0, 1, 2, 3}, retention, triggerChecker1.from),
			metricSource.MakeMetricData("super", []float64{0, 1, 2, 3}, retention, triggerChecker1.from),
		})
		fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric1, metric2}, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		dataBase.EXPECT().RemoveMetricsValues([]string{metric1, metric2}, gomock.Any())
		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), true).Return(nil)
		checkData, err := triggerChecker1.checkTrigger()
		So(err, ShouldResemble, ErrTriggerHasSameMetricNames{names: []string{"super"}})
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				"super": {
					EventTimestamp: 3617,
					State:          moira.StateOK,
					Suppressed:     false,
					Timestamp:      3647,
					Value:          &f,
					Maintenance:    0,
				},
			},
			Score:          0,
			State:          moira.StateNODATA,
			Timestamp:      3667,
			EventTimestamp: 0,
			Suppressed:     false,
			Message:        "",
		})
	})

	Convey("No data too long and ttlState is delete", t, func() {
		triggerChecker.from = 4217
		triggerChecker.until = 4267
		triggerChecker.ttlState = moira.TTLStateDEL
		lastCheck.Timestamp = 4267

		source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)})
		fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-3600)
		dataBase.EXPECT().RemovePatternsMetrics(triggerChecker.trigger.Patterns).Return(nil)

		checkData, err := triggerChecker.checkTrigger()
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics:                      make(map[string]moira.MetricState),
			Timestamp:                    triggerChecker.until,
			State:                        moira.StateOK,
			Score:                        0,
			LastSuccessfulCheckTimestamp: 0,
		})
	})
}

func TestHandleTriggerCheck(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	ttlState := moira.TTLStateNODATA

	Convey("Handle trigger was not successful checked and no error", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       0,
			ttlState:  ttlState,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger, TTLState: &ttlState},
			lastCheck: &moira.CheckData{
				Timestamp: 0,
				State:     moira.StateNODATA,
			},
		}
		checkData := moira.CheckData{
			State:     moira.StateOK,
			Timestamp: time.Now().Unix(),
		}
		actual, err := triggerChecker.handleCheckResult(checkData, nil)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, moira.CheckData{
			State:                        moira.StateOK,
			Timestamp:                    time.Now().Unix(),
			LastSuccessfulCheckTimestamp: time.Now().Unix(),
		})
	})

	Convey("Handle error no metrics", t, func() {
		Convey("TTL is 0", func() {
			triggerChecker := TriggerChecker{
				triggerID: "SuperId",
				database:  dataBase,
				logger:    logger,
				ttl:       0,
				ttlState:  ttlState,
				trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger, TTLState: &ttlState},
				lastCheck: &moira.CheckData{
					Timestamp: 0,
					State:     moira.StateNODATA,
				},
			}
			checkData := moira.CheckData{
				State:                        moira.StateNODATA,
				Timestamp:                    time.Now().Unix(),
				Message:                      "Trigger has no metrics, check your target",
				LastSuccessfulCheckTimestamp: time.Now().Unix(),
			}
			actual, err := triggerChecker.handleCheckResult(checkData, ErrTriggerHasNoMetrics{})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, checkData)
		})

		Convey("TTL is not 0", func() {
			triggerChecker := TriggerChecker{
				triggerID: "SuperId",
				database:  dataBase,
				logger:    logger,
				ttl:       60,
				trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
				ttlState:  moira.TTLStateNODATA,
				lastCheck: &moira.CheckData{
					Timestamp:                    0,
					State:                        moira.StateNODATA,
					LastSuccessfulCheckTimestamp: 0,
				},
			}
			var interval int64 = 24
			checkData := moira.CheckData{
				State:     moira.StateOK,
				Timestamp: time.Now().Unix(),
			}
			event := &moira.NotificationEvent{
				IsTriggerEvent:   true,
				Timestamp:        checkData.Timestamp,
				TriggerID:        triggerChecker.triggerID,
				OldState:         moira.StateNODATA,
				State:            moira.StateNODATA,
				MessageEventInfo: &moira.EventInfo{Interval: &interval},
			}

			dataBase.EXPECT().PushNotificationEvent(event, true).Return(nil)
			actual, err := triggerChecker.handleCheckResult(checkData, ErrTriggerHasNoMetrics{})
			expected := moira.CheckData{
				State:                        moira.StateNODATA,
				Timestamp:                    checkData.Timestamp,
				EventTimestamp:               checkData.Timestamp,
				Message:                      "Trigger has no metrics, check your target",
				LastSuccessfulCheckTimestamp: 0,
			}
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})
	})
	Convey("Handle trigger has only wildcards without metrics in last state", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       60,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
			ttlState:  moira.TTLStateERROR,
			lastCheck: &moira.CheckData{
				Timestamp:                    time.Now().Unix(),
				State:                        moira.StateOK,
				LastSuccessfulCheckTimestamp: time.Now().Unix(),
			},
		}
		checkData := moira.CheckData{
			State:     moira.StateOK,
			Timestamp: time.Now().Unix(),
		}

		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), true).Return(nil)
		actual, err := triggerChecker.handleCheckResult(checkData, ErrTriggerHasOnlyWildcards{})
		expected := moira.CheckData{
			State:                        moira.StateERROR,
			Timestamp:                    checkData.Timestamp,
			EventTimestamp:               checkData.Timestamp,
			Message:                      "Trigger never received metrics",
			LastSuccessfulCheckTimestamp: 0,
		}
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})

	Convey("Handle trigger has only wildcards with metrics in last state", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       60,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
			ttlState:  moira.TTLStateNODATA,
			lastCheck: &moira.CheckData{
				Timestamp:                    time.Now().Unix(),
				State:                        moira.StateOK,
				LastSuccessfulCheckTimestamp: time.Now().Unix(),
			},
		}
		checkData := moira.CheckData{
			Metrics: map[string]moira.MetricState{
				"123": {},
			},
			State:                        moira.StateOK,
			Timestamp:                    time.Now().Unix(),
			LastSuccessfulCheckTimestamp: 0,
		}

		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), true).Return(nil)
		actual, err := triggerChecker.handleCheckResult(checkData, ErrTriggerHasOnlyWildcards{})
		expected := moira.CheckData{
			Metrics:                      checkData.Metrics,
			State:                        moira.StateNODATA,
			Timestamp:                    checkData.Timestamp,
			EventTimestamp:               checkData.Timestamp,
			Message:                      "Trigger never received metrics",
			LastSuccessfulCheckTimestamp: 0,
		}
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})

	Convey("Handle trigger has only wildcards and ttlState is OK", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       60,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
			ttlState:  moira.TTLStateOK,
			lastCheck: &moira.CheckData{
				Timestamp:                    time.Now().Unix(),
				State:                        moira.StateOK,
				LastSuccessfulCheckTimestamp: 0,
			},
		}
		checkData := moira.CheckData{
			Metrics:                      map[string]moira.MetricState{},
			State:                        moira.StateOK,
			Timestamp:                    time.Now().Unix(),
			LastSuccessfulCheckTimestamp: 0,
		}

		actual, err := triggerChecker.handleCheckResult(checkData, ErrTriggerHasOnlyWildcards{})
		expected := moira.CheckData{
			Metrics:                      checkData.Metrics,
			State:                        moira.StateOK,
			Timestamp:                    checkData.Timestamp,
			EventTimestamp:               checkData.Timestamp,
			Message:                      "Trigger never received metrics",
			LastSuccessfulCheckTimestamp: 0,
		}
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})

	Convey("Handle trigger has only wildcards and ttlState is DEL", t, func() {
		now := time.Now().Unix()
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       60,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
			ttlState:  moira.TTLStateDEL,
			lastCheck: &moira.CheckData{
				Timestamp:      now,
				EventTimestamp: now - 3600,
				State:          moira.StateOK,
			},
		}
		checkData := moira.CheckData{
			Metrics:   map[string]moira.MetricState{},
			State:     moira.StateOK,
			Timestamp: now,
		}

		actual, err := triggerChecker.handleCheckResult(checkData, ErrTriggerHasOnlyWildcards{})
		expected := moira.CheckData{
			Metrics:        checkData.Metrics,
			State:          moira.StateOK,
			Timestamp:      now,
			EventTimestamp: now - 3600,
			Message:        "Trigger never received metrics",
		}
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})

	Convey("Handle unknown function in evalExpr", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       60,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
			ttlState:  moira.TTLStateNODATA,
			lastCheck: &moira.CheckData{
				Timestamp:                    time.Now().Unix(),
				State:                        moira.StateOK,
				LastSuccessfulCheckTimestamp: 0,
			},
		}
		checkData := moira.CheckData{
			State:     moira.StateOK,
			Timestamp: time.Now().Unix(),
		}

		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), true).Return(nil)

		actual, err := triggerChecker.handleCheckResult(checkData, local.ErrUnknownFunction{FuncName: "123"})
		expected := moira.CheckData{
			State:                        moira.StateEXCEPTION,
			Timestamp:                    checkData.Timestamp,
			EventTimestamp:               checkData.Timestamp,
			Message:                      "Unknown graphite function: \"123\"",
			LastSuccessfulCheckTimestamp: 0,
		}
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})

	Convey("Handle trigger has same metric names", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       60,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
			ttlState:  moira.TTLStateNODATA,
			lastCheck: &moira.CheckData{
				Timestamp: time.Now().Unix(),
				State:     moira.StateOK,
			},
		}
		checkData := moira.CheckData{
			State:     moira.StateOK,
			Timestamp: time.Now().Unix(),
		}

		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), true).Return(nil)

		actual, err := triggerChecker.handleCheckResult(checkData, ErrTriggerHasSameMetricNames{names: []string{"first", "second"}})
		expected := moira.CheckData{
			State:                        moira.StateERROR,
			Timestamp:                    checkData.Timestamp,
			EventTimestamp:               checkData.Timestamp,
			Message:                      "Several metrics have an identical name: first, second",
			LastSuccessfulCheckTimestamp: 0,
		}
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})

	Convey("Handle trigger error remote trigger response", t, func() {
		now := time.Now()
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       300,
			trigger:   &moira.Trigger{TriggerType: moira.RisingTrigger},
			ttlState:  moira.TTLStateNODATA,
			lastCheck: &moira.CheckData{
				Timestamp:      time.Now().Unix(),
				EventTimestamp: time.Now().Add(-1 * time.Hour).Unix(),
				State:          moira.StateOK,
			},
		}
		Convey("but time since last successful check less than ttl", func() {
			checkData := moira.CheckData{
				State:                        moira.StateOK,
				Timestamp:                    now.Unix(),
				LastSuccessfulCheckTimestamp: now.Add(-1 * time.Minute).Unix(),
			}
			expected := moira.CheckData{
				State:                        moira.StateOK,
				Timestamp:                    now.Unix(),
				EventTimestamp:               time.Now().Add(-1 * time.Hour).Unix(),
				LastSuccessfulCheckTimestamp: now.Add(-1 * time.Minute).Unix(),
			}
			actual, err := triggerChecker.handleCheckResult(checkData, remote.ErrRemoteTriggerResponse{InternalError: fmt.Errorf("pain")})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("and time since last successful check more than ttl", func() {
			checkData := moira.CheckData{
				State:                        moira.StateOK,
				Timestamp:                    now.Unix(),
				LastSuccessfulCheckTimestamp: now.Add(-10 * time.Minute).Unix(),
			}
			expected := moira.CheckData{
				State:                        moira.StateEXCEPTION,
				Message:                      fmt.Sprintf("Remote server unavailable. Trigger is not checked for %d seconds", checkData.Timestamp-checkData.LastSuccessfulCheckTimestamp),
				Timestamp:                    now.Unix(),
				EventTimestamp:               now.Unix(),
				LastSuccessfulCheckTimestamp: now.Add(-10 * time.Minute).Unix(),
			}
			dataBase.EXPECT().PushNotificationEvent(gomock.Any(), true).Return(nil)
			actual, err := triggerChecker.handleCheckResult(checkData, remote.ErrRemoteTriggerResponse{InternalError: fmt.Errorf("pain")})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})
	})

	Convey("Handle additional trigger target has more than one metric data", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			logger:    logger,
			ttl:       60,
			trigger: &moira.Trigger{
				Targets:     []string{"aliasByNode(some.data.*,2)", "aliasByNode(some.more.data.*,2)"},
				TriggerType: moira.RisingTrigger,
			},
			ttlState: moira.TTLStateNODATA,
			lastCheck: &moira.CheckData{
				Timestamp: time.Now().Unix(),
				State:     moira.StateNODATA,
			},
		}
		checkData := moira.CheckData{
			State:     moira.StateNODATA,
			Timestamp: time.Now().Unix(),
		}

		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), true).Return(nil)

		actual, err := triggerChecker.handleCheckResult(checkData, ErrWrongTriggerTargets([]int{2}))
		expected := moira.CheckData{
			State:                        moira.StateERROR,
			Timestamp:                    checkData.Timestamp,
			EventTimestamp:               checkData.Timestamp,
			Message:                      "Target t2 has more than one metric",
			LastSuccessfulCheckTimestamp: 0,
		}
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})
}

func BenchmarkTriggerChecker_Check(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}
	b.ReportAllocs()
	mockCtrl := gomock.NewController(b)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	source := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	defer mockCtrl.Finish()

	var retention int64 = 10
	var warnValue float64 = 10
	var errValue float64 = 20
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"

	var ttl int64 = 30

	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
		source:    source,
		logger:    logger,
		config: &Config{
			MetricsTTLSeconds: 10,
		},
		metrics:  metrics.ConfigureCheckerMetrics(metrics.NewDummyRegistry(), "checker", false).LocalMetrics,
		from:     17,
		until:    67,
		ttl:      ttl,
		ttlState: moira.TTLStateNODATA,
		trigger: &moira.Trigger{
			Name:        "Super trigger",
			ErrorValue:  &errValue,
			WarnValue:   &warnValue,
			TriggerType: moira.RisingTrigger,
			Targets:     []string{pattern},
			Patterns:    []string{pattern},
		},
		lastCheck: &moira.CheckData{
			State:     moira.StateOK,
			Timestamp: 57,
			Metrics: map[string]moira.MetricState{
				metric: {
					State:     moira.StateOK,
					Timestamp: 26,
				},
			},
		},
	}
	lastValue := float64(4)
	eventMetrics := map[string]moira.MetricState{
		metric: {
			EventTimestamp: 17,
			State:          moira.StateOK,
			Suppressed:     false,
			Timestamp:      57,
			Value:          &lastValue,
		},
	}

	lastCheck := moira.CheckData{
		Metrics:                      eventMetrics,
		State:                        moira.StateOK,
		Timestamp:                    triggerChecker.until,
		EventTimestamp:               0,
		Score:                        0,
		LastSuccessfulCheckTimestamp: triggerChecker.until,
	}

	dataBase.EXPECT().RemoveMetricsValues([]string{metric}, int64(57)).Return(nil).AnyTimes()
	source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}).AnyTimes()
	fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil).AnyTimes()
	dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil).AnyTimes()
	for n := 0; n < b.N; n++ {
		err := triggerChecker.Check()
		if err != nil {
			b.Errorf("Check() returned error: %w", err)
		}
	}
}
