package checker

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker/metrics/conversion"
	"github.com/moira-alert/moira/expression"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metrics"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
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
	metricName := "main.metric"
	metricT1 := metricSource.MetricData{
		Name:      "main.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{1, math.NaN(), 3, 4, math.NaN()},
	}
	metricT2 := metricSource.MetricData{
		Name:      "main.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{math.NaN(), 4, 3, 2, 1},
	}
	metrics := map[string]metricSource.MetricData{
		"t1": metricT1,
		"t2": metricT2,
	}
	metricLastState := moira.MetricState{
		Maintenance: 11111,
		Suppressed:  true,
	}

	Convey("Checkpoint more than valueTimestamp", t, func() {
		metricState, err := triggerChecker.getMetricDataState(metricName, metrics, metricLastState, 37, 47)
		So(err, ShouldBeNil)
		So(metricState, ShouldBeNil)
	})

	Convey("Checkpoint lover than valueTimestamp", t, func() {
		Convey("Has all value by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(metricName, metrics, metricLastState, 42, 27)
			So(err, ShouldBeNil)
			So(metricState, ShouldResemble, &moira.MetricState{
				State:          moira.StateOK,
				Timestamp:      42,
				Values:         map[string]float64{"t1": 3, "t2": 3},
				Maintenance:    metricLastState.Maintenance,
				Suppressed:     metricLastState.Suppressed,
				EventTimestamp: 0,
			})
		})

		Convey("No value in main metric data by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(metricName, metrics, metricLastState, 66, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("IsAbsent in main metric data by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(metricName, metrics, metricLastState, 29, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("No value in additional metric data by eventTimestamp step", func() {
			metricState, err := triggerChecker.getMetricDataState(metricName, metrics, metricLastState, 26, 11)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		metricState, err := triggerChecker.getMetricDataState(metricName, metrics, metricLastState, 42, 27)
		So(err.Error(), ShouldResemble, "error value and warning value can not be empty")
		So(metricState, ShouldBeNil)
	})
}

func TestTriggerChecker_PrepareMetrics(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	Convey("Prepare metrics for check:", t, func() {
		triggerChecker := TriggerChecker{
			triggerID: "ID",
			logger:    logger,
			from:      0,
			until:     60,
			lastCheck: &moira.CheckData{},
			trigger: &moira.Trigger{
				AloneMetrics: map[string]bool{},
			},
		}
		Convey("last check has no metrics", func() {
			Convey("fetched metrics is empty", func() {
				prepared, alone, err := triggerChecker.prepareMetrics(map[string][]metricSource.MetricData{})
				So(prepared, ShouldHaveLength, 0)
				So(alone, ShouldHaveLength, 0)
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has metrics", func() {
				triggerChecker.trigger.AloneMetrics = map[string]bool{"t1": true}
				fetched := map[string][]metricSource.MetricData{
					"t1": []metricSource.MetricData{
						*metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0),
					},
				}
				prepared, alone, err := triggerChecker.prepareMetrics(fetched)
				So(prepared, ShouldHaveLength, 0)
				So(alone, ShouldHaveLength, 1)
				So(err, ShouldBeNil)
			})

			Convey("fetched metrics has duplicate metrics", func() {
				triggerChecker.trigger.AloneMetrics = map[string]bool{"t1": true}
				fetched := map[string][]metricSource.MetricData{
					"t1": []metricSource.MetricData{
						*metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0),
						*metricSource.MakeMetricData("123", []float64{4, 5, 6}, 10, 0),
					},
				}
				prepared, alone, err := triggerChecker.prepareMetrics(fetched)
				So(prepared, ShouldHaveLength, 0)
				So(alone, ShouldResemble, map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0)})
				So(err, ShouldResemble, ErrTriggerHasSameMetricNames{duplicates: map[string][]string{"t1": []string{"123"}}})
			})

			Convey("Targets have different metrics", func() {
				fetched := map[string][]metricSource.MetricData{
					"t1": []metricSource.MetricData{
						*metricSource.MakeMetricData("first.metric", []float64{1, 2, 3}, 10, 0),
						*metricSource.MakeMetricData("second.metric", []float64{4, 5, 6}, 10, 0),
						*metricSource.MakeMetricData("third.metric", []float64{4, 5, 6}, 10, 0),
					},
					"t2": []metricSource.MetricData{
						*metricSource.MakeMetricData("second.metric", []float64{4, 5, 6}, 10, 0),
						*metricSource.MakeMetricData("third.metric", []float64{4, 5, 6}, 10, 0),
					},
				}
				prepared, alone, err := triggerChecker.prepareMetrics(fetched)
				So(prepared, ShouldHaveLength, 3)
				So(prepared["first.metric"], ShouldNotBeNil)
				So(prepared["first.metric"], ShouldHaveLength, 2)
				So(prepared["second.metric"], ShouldNotBeNil)
				So(prepared["second.metric"], ShouldHaveLength, 2)
				So(prepared["third.metric"], ShouldNotBeNil)
				So(prepared["third.metric"], ShouldHaveLength, 2)
				So(alone, ShouldBeEmpty)
				So(err, ShouldBeNil)
			})
		})

		Convey("last check has metrics", func() {
			triggerChecker.lastCheck = &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"first":  {Values: map[string]float64{"t1": 0}},
					"second": {Values: map[string]float64{"t1": 0}},
					"third":  {Values: map[string]float64{"t1": 0}},
				}}
			Convey("last check has aloneMetrics", func() {
				triggerChecker.trigger.AloneMetrics = map[string]bool{"t2": true}
				triggerChecker.lastCheck = &moira.CheckData{
					MetricsToTargetRelation: map[string]string{"t2": "alone"},
					Metrics: map[string]moira.MetricState{
						"first":  {Values: map[string]float64{"t1": 0, "t2": 0}},
						"second": {Values: map[string]float64{"t1": 0, "t2": 0}},
						"third":  {Values: map[string]float64{"t1": 0, "t2": 0}},
					}}
				Convey("fetched metrics is empty", func() {
					triggerChecker.trigger.AloneMetrics = map[string]bool{"t2": true}
					prepared, alone, err := triggerChecker.prepareMetrics(map[string][]metricSource.MetricData{})
					So(prepared, ShouldHaveLength, 3)
					for _, actualMetricData := range prepared["t1"] {
						So(actualMetricData.Values, ShouldHaveLength, 1)
						So(actualMetricData.StepTime, ShouldResemble, int64(60))
						So(actualMetricData.StartTime, ShouldResemble, int64(0))
						So(actualMetricData.StopTime, ShouldResemble, int64(60))
					}
					So(alone, ShouldHaveLength, 1)
					aloneMetric := alone["t2"]
					So(aloneMetric.Name, ShouldEqual, "alone")
					So(aloneMetric.Values, ShouldHaveLength, 1)
					So(aloneMetric.StepTime, ShouldResemble, int64(60))
					So(aloneMetric.StartTime, ShouldResemble, int64(0))
					So(aloneMetric.StopTime, ShouldResemble, int64(60))

					So(err, ShouldBeNil)

				})
			})
			Convey("fetched metrics is empty", func() {
				prepared, alone, err := triggerChecker.prepareMetrics(map[string][]metricSource.MetricData{})
				So(prepared, ShouldHaveLength, 3)
				for _, actualMetricData := range prepared["t1"] {
					So(actualMetricData.Values, ShouldHaveLength, 1)
					So(actualMetricData.StepTime, ShouldResemble, int64(60))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(alone, ShouldBeEmpty)
				So(err, ShouldBeNil)
			})
			Convey("fetched metrics has only wildcards, step is 0", func() {
				prepared, alone, err := triggerChecker.prepareMetrics(
					map[string][]metricSource.MetricData{
						"t1": []metricSource.MetricData{
							metricSource.MetricData{Name: "wildcard",
								Wildcard: true,
							},
						},
					})
				So(prepared, ShouldHaveLength, 3)
				for _, actualMetricData := range prepared["t1"] {
					So(actualMetricData.Values, ShouldHaveLength, 1)
					So(actualMetricData.StepTime, ShouldResemble, int64(60))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(alone, ShouldBeEmpty)
				So(err, ShouldBeNil)
			})
			Convey("fetched metrics has only wildcards, step is 10", func() {
				prepared, alone, err := triggerChecker.prepareMetrics(
					map[string][]metricSource.MetricData{
						"t1": []metricSource.MetricData{
							metricSource.MetricData{
								Name:     "wildcard",
								Wildcard: true,
								StepTime: 10,
							},
						},
					})
				So(prepared, ShouldHaveLength, 3)
				for _, actualMetricData := range prepared["t1"] {
					So(actualMetricData.Values, ShouldHaveLength, 6)
					So(actualMetricData.StepTime, ShouldResemble, int64(10))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(alone, ShouldBeEmpty)
				So(err, ShouldBeNil)
			})
			Convey("fetched metrics has one of last check metrics", func() {
				prepared, alone, err := triggerChecker.prepareMetrics(
					map[string][]metricSource.MetricData{
						"t1": []metricSource.MetricData{
							*metricSource.MakeMetricData("first", []float64{1, 2, 3, 4, 5, 6}, 10, 0),
						},
					})
				So(prepared, ShouldHaveLength, 3)
				for _, actualMetricData := range prepared["t1"] {
					So(actualMetricData.Values, ShouldHaveLength, 6)
					So(actualMetricData.StepTime, ShouldResemble, int64(10))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(alone, ShouldBeEmpty)
				So(err, ShouldBeNil)
			})
			Convey("fetched metrics has one of last check metrics and one new", func() {
				prepared, alone, err := triggerChecker.prepareMetrics(
					map[string][]metricSource.MetricData{
						"t1": []metricSource.MetricData{
							*metricSource.MakeMetricData("first", []float64{1, 2, 3, 4, 5, 6}, 10, 0),
							*metricSource.MakeMetricData("fourth", []float64{7, 8, 9, 1, 2, 3}, 10, 0),
						},
					})
				So(prepared, ShouldHaveLength, 4)
				for _, actualMetricData := range prepared["t1"] {
					So(actualMetricData.Values, ShouldHaveLength, 6)
					So(actualMetricData.StepTime, ShouldResemble, int64(10))
					So(actualMetricData.StartTime, ShouldResemble, int64(0))
					So(actualMetricData.StopTime, ShouldResemble, int64(60))
				}
				So(alone, ShouldBeEmpty)
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
		lastCheck: &moira.CheckData{},
	}

	maintenance := int64(11111)
	suppressed := true
	metricData1 := metricSource.MetricData{
		Name:      "main.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{1, math.NaN(), 3, 4, math.NaN()},
	}
	metricData2 := metricSource.MetricData{
		Name:      "main.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{1, 2, 3, 4, 5},
	}
	addMetricData := metricSource.MetricData{
		Name:      "additional.metric",
		StartTime: triggerChecker.from,
		StopTime:  triggerChecker.until,
		StepTime:  10,
		Values:    []float64{5, 4, 3, 2, 1},
	}

	metricsState1 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      17,
		Values:         map[string]float64{"t1": 1, "t2": 5},
		Value:          nil,
		Maintenance:    maintenance,
		Suppressed:     suppressed,
		EventTimestamp: 0,
	}

	metricsState2 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      27,
		Values:         map[string]float64{"t1": 2, "t2": 4},
		Value:          nil,
		Maintenance:    maintenance,
		Suppressed:     suppressed,
		EventTimestamp: 0,
	}

	metricsState3 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      37,
		Values:         map[string]float64{"t1": 3, "t2": 3},
		Value:          nil,
		Maintenance:    maintenance,
		Suppressed:     suppressed,
		EventTimestamp: 0,
	}

	metricsState4 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      47,
		Values:         map[string]float64{"t1": 4, "t2": 2},
		Value:          nil,
		Maintenance:    maintenance,
		Suppressed:     suppressed,
		EventTimestamp: 0,
	}

	metricsState5 := moira.MetricState{
		State:          moira.StateOK,
		Timestamp:      57,
		Values:         map[string]float64{"t1": 5, "t2": 1},
		Value:          nil,
		Maintenance:    maintenance,
		Suppressed:     suppressed,
		EventTimestamp: 0,
	}

	Convey("ValueTimestamp covers all metric range", t, func() {
		triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
			"main.metric": moira.MetricState{
				Maintenance:    11111,
				Suppressed:     true,
				EventTimestamp: 11,
			},
		}
		Convey("Metric has all valid values", func() {
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData})
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("Metric has invalid values", func() {
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData1, "t2": addMetricData})
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState3, metricsState4})
		})

		Convey("Until + stepTime covers last value", func() {
			triggerChecker.until = 56
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData})
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})
	})

	triggerChecker.until = 67

	Convey("ValueTimestamp don't covers begin of metric data", t, func() {
		Convey("Exclude 1 first element", func() {
			triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
				"main.metric": moira.MetricState{
					Maintenance:    11111,
					Suppressed:     true,
					EventTimestamp: 22,
				},
			}
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData})
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude 2 first elements", func() {
			triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
				"main.metric": moira.MetricState{
					Maintenance:    11111,
					Suppressed:     true,
					EventTimestamp: 27,
				},
			}
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData})
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude last element", func() {
			triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
				"main.metric": moira.MetricState{
					Maintenance:    11111,
					Suppressed:     true,
					EventTimestamp: 11,
				},
			}
			triggerChecker.until = 47
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData})
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4})
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
			"main.metric": moira.MetricState{
				Maintenance:    11111,
				Suppressed:     true,
				EventTimestamp: 11,
			},
		}
		triggerChecker.until = 47
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData})
		So(err.Error(), ShouldResemble, "error value and warning value can not be empty")
		So(metricStates, ShouldBeEmpty)
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
	metricName := "main.metric"
	Convey("No TTL", t, func() {
		triggerChecker := TriggerChecker{}
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricName, metricLastState)
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
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricName, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})
		Convey("2", func() {
			metricLastState.Timestamp = 401
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricName, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})
	})

	metricLastState.Timestamp = 399
	triggerChecker.ttlState = moira.TTLStateDEL

	Convey("TTLState is DEL and has EventTimeStamp", t, func() {
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricName, metricLastState)
		So(needToDeleteMetric, ShouldBeTrue)
		So(currentState, ShouldBeNil)
	})

	Convey("Has new metricState", t, func() {
		Convey("TTLState is DEL, but no EventTimestamp", func() {
			metricLastState.EventTimestamp = 0
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricName, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       moira.StateNODATA,
				Timestamp:   triggerChecker.lastCheck.Timestamp,
				Values:      map[string]float64{},
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})

		Convey("TTLState is OK and no EventTimestamp", func() {
			metricLastState.EventTimestamp = 0
			triggerChecker.ttlState = moira.TTLStateOK
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricName, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       triggerChecker.ttlState.ToMetricState(),
				Timestamp:   triggerChecker.lastCheck.Timestamp,
				Values:      map[string]float64{},
				Maintenance: metricLastState.Maintenance,
				Suppressed:  metricLastState.Suppressed,
			})
		})

		Convey("TTLState is OK and has EventTimestamp", func() {
			metricLastState.EventTimestamp = 111
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricName, metricLastState)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldResemble, &moira.MetricState{
				State:       triggerChecker.ttlState.ToMetricState(),
				Timestamp:   triggerChecker.lastCheck.Timestamp,
				Values:      map[string]float64{},
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
		var metricsTTL int64 = 3600
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
			config:    &Config{},
			metrics:   checkerMetrics.LocalMetrics,
			from:      17,
			until:     67,
			ttl:       ttl,
			ttlState:  moira.TTLStateNODATA,
			trigger: &moira.Trigger{
				Name:         "Super trigger",
				ErrorValue:   &errValue,
				WarnValue:    &warnValue,
				TriggerType:  moira.RisingTrigger,
				Targets:      []string{pattern},
				Patterns:     []string{pattern},
				AloneMetrics: map[string]bool{"t1": true},
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
				Metrics:                 triggerChecker.lastCheck.Metrics,
				State:                   moira.StateOK,
				Timestamp:               triggerChecker.until,
				EventTimestamp:          triggerChecker.until,
				Score:                   0,
				Message:                 "",
				MetricsToTargetRelation: map[string]string{},
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
					MetricsToTargetRelation:      map[string]string{},
				}

				gomock.InOrder(
					source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(nil, unknownFunctionExc),
					dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil),
					dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil),
				)
				err := triggerChecker.Check()
				So(err, ShouldBeNil)
			})

			Convey("Switch state to OK. Event should be created", func() {
				triggerChecker.lastCheck.State = moira.StateEXCEPTION
				triggerChecker.lastCheck.EventTimestamp = 67
				triggerChecker.lastCheck.LastSuccessfulCheckTimestamp = triggerChecker.until
				eventMetrics := map[string]moira.MetricState{
					metric: {
						EventTimestamp: 17,
						State:          moira.StateOK,
						Suppressed:     false,
						Timestamp:      57,
						Values:         map[string]float64{"t1": 4},
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
					MetricsToTargetRelation:      map[string]string{"t1": "super.puper.metric"},
				}
				gomock.InOrder(
					source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil),
					fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}),
					fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil),
					dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL),
					dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-metricsTTL).Return(nil),
					dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil),
					dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil),
				)
				err := triggerChecker.Check()
				So(err, ShouldBeNil)
			})
		})

		Convey("Trigger switch to Error", func() {
			lastCheck := moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						EventTimestamp:  57,
						State:           moira.StateERROR,
						Timestamp:       57,
						MaintenanceInfo: moira.MaintenanceInfo{},
						Values:          map[string]float64{"t1": 25},
					},
				},
				Score:                        100,
				State:                        moira.StateOK,
				Timestamp:                    triggerChecker.until,
				EventTimestamp:               triggerChecker.until,
				LastSuccessfulCheckTimestamp: triggerChecker.until,
				MetricsToTargetRelation:      map[string]string{"t1": "super.puper.metric"},
			}
			event := moira.NotificationEvent{
				IsTriggerEvent: false,
				TriggerID:      triggerChecker.triggerID,
				State:          moira.StateERROR,
				OldState:       moira.StateOK,
				Timestamp:      57,
				Metric:         metric,
				Values:         map[string]float64{"t1": 25},
			}

			gomock.InOrder(
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil),
				fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{
					*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 25}, retention, triggerChecker.from),
				}),
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil),
				dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL),
				dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-metricsTTL).Return(nil),
				dataBase.EXPECT().PushNotificationEvent(&event, true).Return(nil),
				dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil),
			)
			err := triggerChecker.Check()
			So(err, ShouldBeNil)
		})
		Convey("Duplicate error", func() {
			lastCheck := moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						EventTimestamp:  17,
						State:           moira.StateOK,
						Timestamp:       57,
						MaintenanceInfo: moira.MaintenanceInfo{},
						Values:          map[string]float64{"t1": 4},
					},
				},
				MetricsToTargetRelation:      map[string]string{"t1": "super.puper.metric"},
				Score:                        100,
				State:                        moira.StateERROR,
				Timestamp:                    triggerChecker.until,
				EventTimestamp:               triggerChecker.until,
				LastSuccessfulCheckTimestamp: triggerChecker.until,
				Message:                      "Targets have metrics with identical name: t1:super.puper.metric; ",
			}
			event := moira.NotificationEvent{
				IsTriggerEvent: true,
				TriggerID:      triggerChecker.triggerID,
				State:          moira.StateERROR,
				OldState:       moira.StateOK,
				Timestamp:      67,
				Metric:         triggerChecker.trigger.Name,
			}

			dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
			dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-metricsTTL).Return(nil)
			source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{
				*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from),
				*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from),
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
		State:     moira.StateNODATA,
		Timestamp: 66,
	}
	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
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

	aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}
	triggerChecker.lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics)
	metricsToCheck := map[string]map[string]metricSource.MetricData{}
	checkData := newCheckData(&lastCheck, triggerChecker.until)

	Convey("First Event, NODATA - OK is ignored", t, func() {
		triggerChecker.trigger.MuteNewMetrics = true
		newCheckData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData)
		So(err, ShouldBeNil)
		So(newCheckData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      time.Now().Unix(),
					EventTimestamp: time.Now().Unix(),
					State:          moira.StateOK,
					Value:          nil,
					Values:         nil,
				},
			},
			MetricsToTargetRelation: map[string]string{"t1": metric},
			Timestamp:               triggerChecker.until,
			State:                   moira.StateNODATA,
			Score:                   0,
		})
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
		State:     moira.StateNODATA,
		Timestamp: 66,
	}
	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
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
		aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}
		lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics)
		checkData := newCheckData(&lastCheck, triggerChecker.until)
		metricsToCheck := map[string]map[string]metricSource.MetricData{}
		dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
			TriggerID: triggerChecker.triggerID,
			Timestamp: 3617,
			State:     moira.StateOK,
			OldState:  moira.StateNODATA,
			Metric:    metric,
			Values:    map[string]float64{"t1": 0},
			Message:   nil}, true).Return(nil)
		checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData)
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      3657,
					EventTimestamp: 3617,
					State:          moira.StateOK,
					Value:          nil,
					Values:         map[string]float64{"t1": 4},
				},
			},
			MetricsToTargetRelation: map[string]string{"t1": metric},
			Timestamp:               triggerChecker.until,
			State:                   moira.StateNODATA,
			Score:                   0,
		})
	})

	lastCheck = moira.CheckData{
		Metrics: map[string]moira.MetricState{
			metric: {
				Timestamp:      3647,
				EventTimestamp: 3607,
				State:          moira.StateOK,
				Values:         map[string]float64{"t1": 3},
			},
		},
		State:     moira.StateOK,
		Timestamp: 3655,
	}

	Convey("Last check is not empty", t, func() {
		aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}
		lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics)
		checkData := newCheckData(&lastCheck, triggerChecker.until)
		metricsToCheck := map[string]map[string]metricSource.MetricData{}

		checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData)
		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      3657,
					EventTimestamp: 3607,
					State:          moira.StateOK,
					Value:          nil,
					Values:         map[string]float64{"t1": 4},
				},
			},
			MetricsToTargetRelation: map[string]string{"t1": metric},
			Timestamp:               triggerChecker.until,
			State:                   moira.StateOK,
			Score:                   0,
		})
	})

	Convey("No data too long", t, func() {
		triggerChecker.from = 4217
		triggerChecker.until = 4267
		lastCheck.Timestamp = 4267
		dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
			TriggerID: triggerChecker.triggerID,
			Timestamp: lastCheck.Timestamp,
			State:     moira.StateNODATA,
			OldState:  moira.StateOK,
			Metric:    metric,
			Values:    map[string]float64{},
			Message:   nil}, true).Return(nil)
		aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)}
		lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics)
		checkData := newCheckData(&lastCheck, triggerChecker.until)
		metricsToCheck := map[string]map[string]metricSource.MetricData{}

		checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData)

		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      lastCheck.Timestamp,
					EventTimestamp: lastCheck.Timestamp,
					State:          moira.StateNODATA,
					Values:         map[string]float64{},
				},
			},
			MetricsToTargetRelation: map[string]string{"t1": "super.puper.metric"},
			Timestamp:               triggerChecker.until,
			State:                   moira.StateOK,
			Score:                   0,
		})
	})

	Convey("No data too long and ttlState is delete", t, func() {
		triggerChecker.from = 4217
		triggerChecker.until = 4267
		triggerChecker.ttlState = moira.TTLStateDEL
		lastCheck.Timestamp = 4267

		dataBase.EXPECT().RemovePatternsMetrics(triggerChecker.trigger.Patterns).Return(nil)

		aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)}
		lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics)
		checkData := newCheckData(&lastCheck, triggerChecker.until)
		metricsToCheck := map[string]map[string]metricSource.MetricData{}

		checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData)

		So(err, ShouldBeNil)
		So(checkData, ShouldResemble, moira.CheckData{
			Metrics:                      make(map[string]moira.MetricState),
			Timestamp:                    triggerChecker.until,
			State:                        moira.StateOK,
			Score:                        0,
			LastSuccessfulCheckTimestamp: 0,
			MetricsToTargetRelation:      map[string]string{"t1": metric},
		})
	})
}

func TestTriggerChecker_Check(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	source := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	defer mockCtrl.Finish()

	var retention int64 = 10
	var metricsTTL int64 = 3600
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
		config:    &Config{},
		metrics:   metrics.ConfigureCheckerMetrics(metrics.NewDummyRegistry(), false).LocalMetrics,
		from:      17,
		until:     67,
		ttl:       ttl,
		ttlState:  moira.TTLStateNODATA,
		trigger: &moira.Trigger{
			Name:         "Super trigger",
			ErrorValue:   &errValue,
			WarnValue:    &warnValue,
			TriggerType:  moira.RisingTrigger,
			Targets:      []string{pattern},
			Patterns:     []string{pattern},
			AloneMetrics: map[string]bool{"t1": true},
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
			MetricsToTargetRelation: map[string]string{"t1": metric},
		},
	}
	eventMetrics := map[string]moira.MetricState{
		metric: {
			EventTimestamp: 17,
			State:          moira.StateOK,
			Suppressed:     false,
			Timestamp:      57,
			Values:         map[string]float64{"t1": 4},
		},
	}

	lastCheck := moira.CheckData{
		Metrics:                      eventMetrics,
		State:                        moira.StateOK,
		Timestamp:                    triggerChecker.until,
		EventTimestamp:               triggerChecker.until,
		Score:                        0,
		LastSuccessfulCheckTimestamp: triggerChecker.until,
		MetricsToTargetRelation:      map[string]string{"t1": metric},
	}

	dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
	dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-metricsTTL).Return(nil)
	source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)})
	fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
	dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil)
	_ = triggerChecker.Check()
}

func BenchmarkTriggerChecker_Check(b *testing.B) {
	b.ReportAllocs()
	mockCtrl := gomock.NewController(b)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	source := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	defer mockCtrl.Finish()

	var retention int64 = 10
	var metricsTTL int64 = 3600
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
		config:    &Config{},
		metrics:   metrics.ConfigureCheckerMetrics(metrics.NewDummyRegistry(), false).LocalMetrics,
		from:      17,
		until:     67,
		ttl:       ttl,
		ttlState:  moira.TTLStateNODATA,
		trigger: &moira.Trigger{
			Name:         "Super trigger",
			ErrorValue:   &errValue,
			WarnValue:    &warnValue,
			TriggerType:  moira.RisingTrigger,
			Targets:      []string{pattern},
			Patterns:     []string{pattern},
			AloneMetrics: map[string]bool{"t1": true},
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
			MetricsToTargetRelation: map[string]string{"t1": metric},
		},
	}
	eventMetrics := map[string]moira.MetricState{
		metric: {
			EventTimestamp: 17,
			State:          moira.StateOK,
			Suppressed:     false,
			Timestamp:      57,
			Values:         map[string]float64{"t1": 4},
		},
	}

	lastCheck := moira.CheckData{
		Metrics:                      eventMetrics,
		State:                        moira.StateOK,
		Timestamp:                    triggerChecker.until,
		EventTimestamp:               triggerChecker.until,
		Score:                        0,
		LastSuccessfulCheckTimestamp: triggerChecker.until,
		MetricsToTargetRelation:      map[string]string{"t1": metric},
	}

	dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL).AnyTimes()
	dataBase.EXPECT().RemoveMetricsValues([]string{metric}, triggerChecker.until-metricsTTL).Return(nil).AnyTimes()
	source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}).AnyTimes()
	fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil).AnyTimes()
	dataBase.EXPECT().SetTriggerLastCheck(triggerChecker.triggerID, &lastCheck, triggerChecker.trigger.IsRemote).Return(nil).AnyTimes()
	for n := 0; n < b.N; n++ {
		err := triggerChecker.Check()
		if err != nil {
			b.Errorf("Check() returned error: %w", err)
		}
	}
}

func TestGetExpressionValues(t *testing.T) {
	Convey("Has only main metric data", t, func() {
		metricData := metricSource.MetricData{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0.0, math.NaN(), math.NaN(), 3.0, math.NaN()},
		}
		metrics := map[string]metricSource.MetricData{
			"t1": metricData,
		}

		Convey("first value is valid", func() {
			expectedExpression := &expression.TriggerExpression{
				AdditionalTargetsValues: make(map[string]float64),
			}
			expectedValues := map[string]float64{"t1": 0}

			expression, values, noEmptyValues := getExpressionValues(metrics, 17)
			So(noEmptyValues, ShouldBeTrue)
			So(expression, ShouldResemble, expectedExpression)
			So(values, ShouldResemble, expectedValues)
		})
		Convey("last value is empty", func() {
			_, _, noEmptyValues := getExpressionValues(metrics, 67)
			So(noEmptyValues, ShouldBeFalse)
		})
		Convey("value before first value", func() {
			_, _, noEmptyValues := getExpressionValues(metrics, 11)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("value in the middle is empty ", func() {
			_, _, noEmptyValues := getExpressionValues(metrics, 44)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("value in the middle is valid", func() {
			expectedExpression := &expression.TriggerExpression{
				MainTargetValue:         3,
				AdditionalTargetsValues: make(map[string]float64),
			}
			expectedValues := map[string]float64{"t1": 3}

			expression, values, noEmptyValues := getExpressionValues(metrics, 53)
			So(noEmptyValues, ShouldBeTrue)
			So(expression, ShouldResemble, expectedExpression)
			So(values, ShouldResemble, expectedValues)
		})
	})

	Convey("Has additional series", t, func() {
		metricData := metricSource.MetricData{
			Name:      "main",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0.0, math.NaN(), math.NaN(), 3.0, math.NaN()},
		}
		metricDataAdd := metricSource.MetricData{
			Name:      "main",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{4.0, 3.0, math.NaN(), math.NaN(), 0.0},
		}
		metrics := map[string]metricSource.MetricData{
			"t1": metricData,
			"t2": metricDataAdd,
		}

		Convey("t1 value in the middle is empty ", func() {
			_, _, noEmptyValues := getExpressionValues(metrics, 29)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("t1 and t2 values in the middle is empty ", func() {
			_, _, noEmptyValues := getExpressionValues(metrics, 42)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("both first values is valid ", func() {
			expectedValues := map[string]float64{"t1": 0, "t2": 4}

			expression, values, noEmptyValues := getExpressionValues(metrics, 17)
			So(noEmptyValues, ShouldBeTrue)
			So(expression.MainTargetValue, ShouldBeIn, []float64{0, 4})
			So(values, ShouldResemble, expectedValues)
		})
	})
}

func TestTriggerChecker_validateAloneMetrics(t *testing.T) {

	tests := []struct {
		name         string
		trigger      moira.Trigger
		aloneMetrics map[string]metricSource.MetricData
		wantErr      func(actual interface{}, expected ...interface{}) string
	}{
		{
			name: "trigger have one target and metric in this target is alone",
			trigger: moira.Trigger{
				Targets:      []string{"test.target.1.*"},
				AloneMetrics: map[string]bool{},
			},
			aloneMetrics: map[string]metricSource.MetricData{
				"t1": metricSource.MetricData{},
			},
			wantErr: ShouldBeNil,
		},
		{
			name: "trigger have couple targets and actual alone metrics fit expected",
			trigger: moira.Trigger{
				Targets:      []string{"test.target.1.*", "test.target.2.*"},
				AloneMetrics: map[string]bool{"t1": true},
			},
			aloneMetrics: map[string]metricSource.MetricData{
				"t1": metricSource.MetricData{},
			},
			wantErr: ShouldBeNil,
		},
		{
			name: "trigger have one target and metrics in this target are not alone",
			trigger: moira.Trigger{
				Targets:      []string{"test.target.1.*"},
				AloneMetrics: map[string]bool{},
			},
			aloneMetrics: map[string]metricSource.MetricData{},
			wantErr:      ShouldBeNil,
		},
		{
			name: "trigger have couple targets and actual targets have more alone metrics than expected",
			trigger: moira.Trigger{
				Targets:      []string{"test.target.1.*", "test.target.2.*"},
				AloneMetrics: map[string]bool{"t1": true},
			},
			aloneMetrics: map[string]metricSource.MetricData{"t1": metricSource.MetricData{}, "t2": metricSource.MetricData{}},
			wantErr:      ShouldBeError,
		},
		{
			name: "trigger have couple targets and actual targets have less alone metrics than expected",
			trigger: moira.Trigger{
				Targets:      []string{"test.target.1.*", "test.target.2.*"},
				AloneMetrics: map[string]bool{"t1": true, "t2": true},
			},
			aloneMetrics: map[string]metricSource.MetricData{"t1": metricSource.MetricData{}},
			wantErr:      ShouldBeError,
		},
		{
			name: "trigger have couple targets and actual targets have different alone metrics than expected",
			trigger: moira.Trigger{
				Targets:      []string{"test.target.1.*", "test.target.2.*"},
				AloneMetrics: map[string]bool{"t1": true},
			},
			aloneMetrics: map[string]metricSource.MetricData{"t2": metricSource.MetricData{}},
			wantErr:      ShouldBeError,
		},
	}
	Convey("validateAloneMetrics", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				triggerChecker := TriggerChecker{
					trigger: &tt.trigger,
				}
				err := triggerChecker.validateAloneMetrics(tt.aloneMetrics)
				So(err, tt.wantErr)
			})
		}
	})
}
