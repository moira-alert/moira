package checker

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker/metrics/conversion"
	"github.com/moira-alert/moira/expression"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"go.uber.org/mock/gomock"

	"github.com/moira-alert/moira/metrics"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

var defaultLocalClusterKey = moira.MakeClusterKey(moira.GraphiteLocal, "default")

func TestGetMetricDataState(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	var warnValue float64 = 10
	var errValue float64 = 20
	checkerMetrics, _ := metrics.
		ConfigureCheckerMetrics(metrics.NewDummyRegistry(), []moira.ClusterKey{defaultLocalClusterKey}).
		GetCheckMetricsBySource(defaultLocalClusterKey)
	triggerChecker := TriggerChecker{
		logger:  logger,
		metrics: checkerMetrics,
		until:   67,
		from:    17,
		trigger: &moira.Trigger{
			WarnValue:   &warnValue,
			ErrorValue:  &errValue,
			TriggerType: moira.RisingTrigger,
		},
	}
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
		var valueTimestamp int64 = 37
		var checkPoint int64 = 47
		metricState, err := triggerChecker.getMetricDataState(metrics, &metricLastState, &valueTimestamp, &checkPoint, logger)
		So(err, ShouldBeNil)
		So(metricState, ShouldBeNil)
	})

	Convey("Checkpoint lover than valueTimestamp", t, func() {
		Convey("Has all value by eventTimestamp step", func() {
			var valueTimestamp int64 = 42
			var checkPoint int64 = 27
			metricState, err := triggerChecker.getMetricDataState(metrics, &metricLastState, &valueTimestamp, &checkPoint, logger)
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
			var valueTimestamp int64 = 66
			var checkPoint int64 = 11
			metricState, err := triggerChecker.getMetricDataState(metrics, &metricLastState, &valueTimestamp, &checkPoint, logger)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("IsAbsent in main metric data by eventTimestamp step", func() {
			var valueTimestamp int64 = 29
			var checkPoint int64 = 11
			metricState, err := triggerChecker.getMetricDataState(metrics, &metricLastState, &valueTimestamp, &checkPoint, logger)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})

		Convey("No value in additional metric data by eventTimestamp step", func() {
			var valueTimestamp int64 = 26
			var checkPoint int64 = 11
			metricState, err := triggerChecker.getMetricDataState(metrics, &metricLastState, &valueTimestamp, &checkPoint, logger)
			So(err, ShouldBeNil)
			So(metricState, ShouldBeNil)
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		var valueTimestamp int64 = 42
		var checkPoint int64 = 27
		metricState, err := triggerChecker.getMetricDataState(metrics, &metricLastState, &valueTimestamp, &checkPoint, logger)
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
					"t1": {
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
					"t1": {
						*metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0),
						*metricSource.MakeMetricData("123", []float64{4, 5, 6}, 10, 0),
					},
				}
				prepared, alone, err := triggerChecker.prepareMetrics(fetched)
				So(prepared, ShouldHaveLength, 0)
				So(alone, ShouldResemble, map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData("123", []float64{1, 2, 3}, 10, 0)})
				So(err, ShouldResemble, ErrTriggerHasSameMetricNames{duplicates: map[string][]string{"t1": {"123"}}})
			})

			Convey("Targets have different metrics", func() {
				fetched := map[string][]metricSource.MetricData{
					"t1": {
						*metricSource.MakeMetricData("first.metric", []float64{1, 2, 3}, 10, 0),
						*metricSource.MakeMetricData("second.metric", []float64{4, 5, 6}, 10, 0),
						*metricSource.MakeMetricData("third.metric", []float64{4, 5, 6}, 10, 0),
					},
					"t2": {
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

			Convey("Targets with alone metrics do not have metrics", func() {
				fetched := map[string][]metricSource.MetricData{
					"t1": {
						*metricSource.MakeMetricData("first.metric", []float64{1, 2, 3}, 10, 0),
						*metricSource.MakeMetricData("second.metric", []float64{4, 5, 6}, 10, 0),
						*metricSource.MakeMetricData("third.metric", []float64{4, 5, 6}, 10, 0),
					},
					"t2": {},
				}
				triggerChecker.trigger.AloneMetrics = map[string]bool{"t2": true}
				prepared, alone, err := triggerChecker.prepareMetrics(fetched)
				So(err, ShouldResemble, conversion.NewErrEmptyAloneMetricsTarget("t2"))
				So(alone, ShouldBeEmpty)
				So(prepared, ShouldBeEmpty)
			})
		})

		Convey("last check has metrics", func() {
			triggerChecker.lastCheck = &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"first":  {Values: map[string]float64{"t1": 0}},
					"second": {Values: map[string]float64{"t1": 0}},
					"third":  {Values: map[string]float64{"t1": 0}},
				},
			}
			Convey("last check has aloneMetrics", func() {
				triggerChecker.trigger.AloneMetrics = map[string]bool{"t2": true}
				triggerChecker.lastCheck = &moira.CheckData{
					MetricsToTargetRelation: map[string]string{"t2": "alone"},
					Metrics: map[string]moira.MetricState{
						"first":  {Values: map[string]float64{"t1": 0, "t2": 0}},
						"second": {Values: map[string]float64{"t1": 0, "t2": 0}},
						"third":  {Values: map[string]float64{"t1": 0, "t2": 0}},
					},
				}
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
						"t1": {
							{
								Name:     "wildcard",
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
						"t1": {
							{
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
						"t1": {
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
						"t1": {
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
	logger.Level("info") // nolint: errcheck
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
			"main.metric": {
				Maintenance:    11111,
				Suppressed:     true,
				EventTimestamp: 11,
			},
		}

		Convey("Metric has all valid values", func() {
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData}, logger)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("Metric has invalid values", func() {
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData1, "t2": addMetricData}, logger)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState3, metricsState4})
		})

		Convey("Until + stepTime covers last value", func() {
			triggerChecker.until = 56
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData}, logger)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4, metricsState5})
		})
	})

	triggerChecker.until = 67

	Convey("ValueTimestamp don't covers begin of metric data", t, func() {
		Convey("Exclude 1 first element", func() {
			triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
				"main.metric": {
					Maintenance:    11111,
					Suppressed:     true,
					EventTimestamp: 22,
				},
			}
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData}, logger)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState2, metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude 2 first elements", func() {
			triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
				"main.metric": {
					Maintenance:    11111,
					Suppressed:     true,
					EventTimestamp: 27,
				},
			}
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData}, logger)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState3, metricsState4, metricsState5})
		})

		Convey("Exclude last element", func() {
			triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
				"main.metric": {
					Maintenance:    11111,
					Suppressed:     true,
					EventTimestamp: 11,
				},
			}
			triggerChecker.until = 47
			_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData}, logger)
			So(err, ShouldBeNil)
			So(metricStates, ShouldResemble, []moira.MetricState{metricsState1, metricsState2, metricsState3, metricsState4})
		})
	})

	Convey("No warn and error value with default expression", t, func() {
		triggerChecker.lastCheck.Metrics = map[string]moira.MetricState{
			"main.metric": {
				Maintenance:    11111,
				Suppressed:     true,
				EventTimestamp: 11,
			},
		}
		triggerChecker.until = 47
		triggerChecker.trigger.WarnValue = nil
		triggerChecker.trigger.ErrorValue = nil
		_, metricStates, err := triggerChecker.getMetricStepsStates("main.metric", map[string]metricSource.MetricData{"t1": metricData2, "t2": addMetricData}, logger)
		So(err.Error(), ShouldResemble, "error value and warning value can not be empty")
		So(metricStates, ShouldBeEmpty)
	})
}

func TestCheckForNODATA(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	logger.Level("info") // nolint: errcheck
	metricLastState := moira.MetricState{
		EventTimestamp: 11,
		Maintenance:    11111,
		Suppressed:     true,
	}

	Convey("No TTL", t, func() {
		triggerChecker := TriggerChecker{}
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
		So(needToDeleteMetric, ShouldBeFalse)
		So(currentState, ShouldBeNil)
	})

	var ttl int64 = 600

	checkerMetrics, _ := metrics.
		ConfigureCheckerMetrics(metrics.NewDummyRegistry(), []moira.ClusterKey{defaultLocalClusterKey}).
		GetCheckMetricsBySource(defaultLocalClusterKey)
	triggerChecker := TriggerChecker{
		metrics: checkerMetrics,
		logger:  logger,
		ttl:     ttl,
		lastCheck: &moira.CheckData{
			Timestamp: 1000,
		},
	}

	Convey("Last check is resent", t, func() {
		Convey("1", func() {
			metricLastState.Timestamp = 1100
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})

		Convey("2", func() {
			metricLastState.Timestamp = 401
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
			So(needToDeleteMetric, ShouldBeFalse)
			So(currentState, ShouldBeNil)
		})
	})

	metricLastState.Timestamp = 399
	triggerChecker.ttlState = moira.TTLStateDEL

	Convey("TTLState is DEL, has EventTimeStamp, Maintenance metric has expired and will be deleted", t, func() {
		metricLastState.Maintenance = 111
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
		So(needToDeleteMetric, ShouldBeTrue)
		So(currentState, ShouldBeNil)
	})

	Convey("TTLState is DEL, has EventTimeStamp, the metric doesn't have Maintenance and will be deleted", t, func() {
		metricLastState.Maintenance = 0
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
		So(needToDeleteMetric, ShouldBeTrue)
		So(currentState, ShouldBeNil)
	})

	Convey("TTLState is DEL, has EventTimeStamp, but the metric is on Maintenance, so it's not deleted and DeletedButKept = true", t, func() {
		metricLastState.Maintenance = 11111
		needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
		So(needToDeleteMetric, ShouldBeFalse)
		So(currentState, ShouldNotBeNil)
		So(*currentState, ShouldResemble, moira.MetricState{
			Timestamp:      metricLastState.Timestamp,
			EventTimestamp: metricLastState.EventTimestamp,
			Maintenance:    metricLastState.Maintenance,
			Suppressed:     metricLastState.Suppressed,
			DeletedButKept: true,
		})
	})

	Convey("Has new metricState", t, func() {
		Convey("TTLState is DEL, but no EventTimestamp", func() {
			metricLastState.EventTimestamp = 0
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
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
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
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
			needToDeleteMetric, currentState := triggerChecker.checkForNoData(metricLastState, logger)
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
		pattern := "super.puper.pattern" //nolint
		metric := "super.puper.metric"   //nolint
		message := "ooops, metric error"
		metricErr := fmt.Errorf(message)
		messageException := `Unknown graphite function: "WrongFunction"`
		unknownFunctionExc := local.ErrorUnknownFunction(fmt.Errorf(messageException))

		testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC).Unix()

		var ttl int64 = 30

		checkerMetrics, _ := metrics.
			ConfigureCheckerMetrics(metrics.NewDummyRegistry(), []moira.ClusterKey{defaultLocalClusterKey}).
			GetCheckMetricsBySource(defaultLocalClusterKey)
		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			database:  dataBase,
			source:    source,
			logger:    logger,
			config:    &Config{},
			metrics:   checkerMetrics,
			from:      testTime - 5*retention,
			until:     testTime,
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
				Timestamp: testTime - retention,
				Metrics: map[string]moira.MetricState{
					metric: {
						State:     moira.StateOK,
						Timestamp: testTime - 4*retention - 1,
					},
				},
			},
		}

		Convey("Fetch error", func() {
			lastCheck := moira.CheckData{
				Metrics:                 triggerChecker.lastCheck.Metrics,
				State:                   moira.StateEXCEPTION,
				Timestamp:               triggerChecker.until,
				EventTimestamp:          triggerChecker.until,
				Score:                   int64(100000),
				Message:                 metricErr.Error(),
				MetricsToTargetRelation: map[string]string{},
			}

			gomock.InOrder(
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(nil, metricErr),
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					IsTriggerEvent: true,
					TriggerID:      triggerChecker.triggerID,
					State:          moira.StateEXCEPTION,
					OldState:       moira.StateOK,
					Timestamp:      testTime,
					Metric:         triggerChecker.trigger.Name,
				}, true).Return(nil),
				dataBase.EXPECT().SetTriggerLastCheck(
					triggerChecker.triggerID,
					&lastCheck,
					triggerChecker.trigger.ClusterKey(),
				).Return(nil),
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
					Timestamp:      testTime,
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
					dataBase.EXPECT().SetTriggerLastCheck(
						triggerChecker.triggerID,
						&lastCheck,
						triggerChecker.trigger.ClusterKey(),
					).Return(nil),
				)
				err := triggerChecker.Check()
				So(err, ShouldBeNil)
			})

			Convey("Switch state to OK. Event should be created", func() {
				triggerChecker.lastCheck.State = moira.StateEXCEPTION
				triggerChecker.lastCheck.EventTimestamp = testTime
				triggerChecker.lastCheck.LastSuccessfulCheckTimestamp = triggerChecker.until
				eventMetrics := map[string]moira.MetricState{
					metric: {
						EventTimestamp: testTime - 5*retention,
						State:          moira.StateOK,
						Suppressed:     false,
						Timestamp:      testTime - retention,
						Values:         map[string]float64{"t1": 4},
					},
				}

				event := moira.NotificationEvent{
					IsTriggerEvent: true,
					TriggerID:      triggerChecker.triggerID,
					State:          moira.StateOK,
					OldState:       moira.StateEXCEPTION,
					Timestamp:      testTime,
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
					dataBase.EXPECT().SetTriggerLastCheck(
						triggerChecker.triggerID,
						&lastCheck,
						triggerChecker.trigger.ClusterKey(),
					).Return(nil),
				)
				err := triggerChecker.Check()
				So(err, ShouldBeNil)
			})
		})

		Convey("Trigger switch to Error", func() {
			lastCheck := moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						EventTimestamp:  testTime - retention,
						State:           moira.StateERROR,
						Timestamp:       testTime - retention,
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
				Timestamp:      testTime - retention,
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
				dataBase.EXPECT().SetTriggerLastCheck(
					triggerChecker.triggerID,
					&lastCheck,
					triggerChecker.trigger.ClusterKey(),
				).Return(nil),
			)
			err := triggerChecker.Check()
			So(err, ShouldBeNil)
		})

		Convey("Duplicate error", func() {
			lastCheck := moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						EventTimestamp:  testTime - 5*retention,
						State:           moira.StateOK,
						Timestamp:       testTime - retention,
						MaintenanceInfo: moira.MaintenanceInfo{},
						Values:          map[string]float64{"t1": 4},
					},
				},
				MetricsToTargetRelation:      map[string]string{"t1": "super.puper.metric"},
				Score:                        100000,
				State:                        moira.StateEXCEPTION,
				Timestamp:                    triggerChecker.until,
				EventTimestamp:               triggerChecker.until,
				LastSuccessfulCheckTimestamp: triggerChecker.until,
				Message:                      "Targets have metrics with identical name: t1:super.puper.metric; ",
			}
			event := moira.NotificationEvent{
				IsTriggerEvent: true,
				TriggerID:      triggerChecker.triggerID,
				State:          moira.StateEXCEPTION,
				OldState:       moira.StateOK,
				Timestamp:      testTime,
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
			dataBase.EXPECT().SetTriggerLastCheck(
				triggerChecker.triggerID,
				&lastCheck,
				triggerChecker.trigger.ClusterKey(),
			).Return(nil)
			err := triggerChecker.Check()
			So(err, ShouldBeNil)
		})

		Convey("Alone metrics error", func() {
			mockTime := mock_clock.NewMockClock(mockCtrl)
			metricName1 := "test.metric.1"
			metricName2 := "test.metric.2"
			metricNameAlone := "test.metric.alone"
			pattern1 := "target.pattern.1"
			pattern2 := "target.pattern.2"
			pattern3 := "target.pattern.3"
			lastCheck := moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metricName1: {
						EventTimestamp:  testTime - checkPointGap,
						State:           moira.StateNODATA,
						Timestamp:       testTime,
						MaintenanceInfo: moira.MaintenanceInfo{},
					},
					metricName2: {
						EventTimestamp:  testTime - checkPointGap,
						State:           moira.StateNODATA,
						Timestamp:       testTime,
						MaintenanceInfo: moira.MaintenanceInfo{},
					},
				},
				MetricsToTargetRelation:      map[string]string{"t2": metricNameAlone},
				Score:                        2000,
				State:                        moira.StateOK,
				Timestamp:                    triggerChecker.until,
				EventTimestamp:               triggerChecker.until,
				LastSuccessfulCheckTimestamp: triggerChecker.until,
				Message:                      "",
				Clock:                        mockTime,
			}
			expression := "OK"
			triggerChecker.trigger.AloneMetrics = map[string]bool{"t2": true}
			triggerChecker.trigger.Targets = []string{pattern1, pattern2, pattern3}
			triggerChecker.trigger.TriggerType = moira.ExpressionTrigger
			triggerChecker.trigger.Expression = &expression
			triggerChecker.lastCheck = &moira.CheckData{
				Metrics:   map[string]moira.MetricState{},
				State:     moira.StateOK,
				Timestamp: triggerChecker.until - metricsTTL,
				Clock:     mockTime,
			}

			gomock.InOrder(
				source.EXPECT().Fetch(pattern1, triggerChecker.from, triggerChecker.until, false).Return(fetchResult, nil),
				fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{
					*metricSource.MakeMetricData(metricName1, []float64{1, 1, 1, 1, 1}, retention, triggerChecker.from),
				}),
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{metricName1}, nil),

				source.EXPECT().Fetch(pattern2, triggerChecker.from, triggerChecker.until, false).Return(fetchResult, nil),
				fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{
					*metricSource.MakeMetricData(metricNameAlone, []float64{5, 5, 5, 5, 5}, retention, triggerChecker.from),
				}),
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{metricNameAlone}, nil),

				source.EXPECT().Fetch(pattern3, triggerChecker.from, triggerChecker.until, false).Return(fetchResult, nil),
				fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{
					*metricSource.MakeMetricData(metricName2, []float64{2, 2, 2, 2, 2}, retention, triggerChecker.from),
				}),
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{metricName2}, nil),

				dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL),
				dataBase.EXPECT().RemoveMetricsValues([]string{metricName1, metricNameAlone, metricName2}, triggerChecker.until-metricsTTL).Return(nil),

				mockTime.EXPECT().NowUnix().Return(testTime).Times(4),

				dataBase.EXPECT().SetTriggerLastCheck(
					triggerChecker.triggerID,
					&lastCheck,
					triggerChecker.trigger.ClusterKey(),
				).Return(nil),
			)
			err := triggerChecker.Check()
			So(err, ShouldBeNil)
		})
	})
}

func TestCheckWithNoMetrics(t *testing.T) {
	logger, _ := logging.GetLogger("Test")
	metricsToCheck := map[string]map[string]metricSource.MetricData{}

	Convey("given triggerChecker.check is called with empty metric map", t, func() {
		warnValue := float64(10)
		errValue := float64(20)
		pattern := "super.puper.pattern"
		ttl := int64(600)

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
		aloneMetrics := map[string]metricSource.MetricData{}
		checkData := newCheckData(&lastCheck, triggerChecker.until)
		newCheckData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)

		So(err, ShouldBeNil)
		So(newCheckData, ShouldResemble, moira.CheckData{
			Metrics:                 map[string]moira.MetricState{},
			MetricsToTargetRelation: map[string]string{},
			Timestamp:               triggerChecker.until,
			State:                   moira.StateNODATA,
			Score:                   0,
		})
	})
}

func TestIgnoreNodataToOk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("Test")
	logger.Level("info") // nolint: errcheck
	defer mockCtrl.Finish()

	mockTime := mock_clock.NewMockClock(mockCtrl)

	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC).Unix()

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
		Clock:     mockTime,
	}
	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		logger:    logger,
		config:    &Config{},
		from:      testTime - ttl,
		until:     testTime,
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
	triggerChecker.lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
	metricsToCheck := map[string]map[string]metricSource.MetricData{}
	checkData := newCheckData(&lastCheck, triggerChecker.until)

	Convey("First Event, NODATA - OK is ignored", t, func() {
		mockTime.EXPECT().NowUnix().Return(testTime).Times(2)

		triggerChecker.trigger.MuteNewMetrics = true
		newCheckData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
		So(err, ShouldBeNil)
		So(newCheckData, ShouldResemble, moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      testTime,
					EventTimestamp: testTime - checkPointGap,
					State:          moira.StateOK,
					Value:          nil,
					Values:         nil,
				},
			},
			MetricsToTargetRelation: map[string]string{},
			Timestamp:               triggerChecker.until,
			State:                   moira.StateNODATA,
			Score:                   0,
			Clock:                   mockTime,
		})
	})
}

func TestHandleTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockTime := mock_clock.NewMockClock(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	logger.Level("info") // nolint: errcheck
	defer mockCtrl.Finish()

	var metricsTTL int64 = 3600
	var retention int64 = 10
	var warnValue float64 = 10
	var errValue float64 = 20
	pattern := "super.puper.pattern"
	metric := "super.puper.metric"
	var ttl int64 = 600
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC).Unix()

	lastCheck := moira.CheckData{
		Metrics:   make(map[string]moira.MetricState),
		State:     moira.StateNODATA,
		Timestamp: testTime - metricsTTL,
		Clock:     mockTime,
	}

	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
		logger:    logger,
		config:    &Config{},
		from:      testTime - 5*retention,
		until:     testTime,
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

	Convey("Simple mode", t, func() {
		Convey("First Event", func() {
			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}
			lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
			checkData := newCheckData(&lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}

			mockTime.EXPECT().NowUnix().Return(testTime).Times(2)
			dataBase.EXPECT().PushNotificationEvent(
				&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: testTime - 5*retention,
					State:     moira.StateOK,
					OldState:  moira.StateNODATA,
					Metric:    metric,
					Values:    map[string]float64{"t1": 0},
					Message:   nil,
				}, true).Return(nil)

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						Timestamp:      testTime - retention,
						EventTimestamp: testTime - 5*retention,
						State:          moira.StateOK,
						Value:          nil,
						Values:         map[string]float64{"t1": 4},
					},
				},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateNODATA,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		lastCheck = moira.CheckData{
			Metrics: map[string]moira.MetricState{
				metric: {
					Timestamp:      testTime - 2*retention,
					EventTimestamp: testTime - 6*retention,
					State:          moira.StateOK,
					Values:         map[string]float64{"t1": 3},
				},
			},
			State:     moira.StateOK,
			Timestamp: testTime - retention - 2,
			Clock:     mockTime,
		}

		Convey("Last check is not empty", func() {
			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, triggerChecker.from)}
			lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
			checkData := newCheckData(&lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						Timestamp:      testTime - retention,
						EventTimestamp: testTime - 6*retention,
						State:          moira.StateOK,
						Value:          nil,
						Values:         map[string]float64{"t1": 4},
					},
				},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateOK,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		Convey("No data too long", func() {
			triggerChecker.from = testTime + ttl - 5*retention
			triggerChecker.until = testTime + ttl
			lastCheck.Timestamp = testTime + ttl

			dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
				TriggerID: triggerChecker.triggerID,
				Timestamp: lastCheck.Timestamp,
				State:     moira.StateNODATA,
				OldState:  moira.StateOK,
				Metric:    metric,
				Values:    map[string]float64{},
				Message:   nil,
			}, true).Return(nil)
			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)}
			lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
			checkData := newCheckData(&lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
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
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateOK,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		Convey("No data too long and ttlState is delete, the metric is not on Maintenance, so it will be removed", func() {
			triggerChecker.from = testTime + ttl - 5*retention
			triggerChecker.until = testTime + ttl
			triggerChecker.ttlState = moira.TTLStateDEL
			lastCheck.Timestamp = testTime + ttl

			dataBase.EXPECT().RemovePatternsMetrics(triggerChecker.trigger.Patterns).Return(nil)

			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)}
			lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
			checkData := newCheckData(&lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics:                      make(map[string]moira.MetricState),
				Timestamp:                    triggerChecker.until,
				State:                        moira.StateOK,
				Score:                        0,
				LastSuccessfulCheckTimestamp: 0,
				MetricsToTargetRelation:      map[string]string{},
				Clock:                        mockTime,
			})
		})

		metricState := lastCheck.Metrics[metric]
		metricState.Maintenance = testTime + ttl
		lastCheck.Metrics[metric] = metricState

		Convey("No data too long and ttlState is delete, but the metric is on maintenance and DeletedButKept is false, so it won't be deleted", func() {
			triggerChecker.from = testTime + ttl - 5*retention
			triggerChecker.until = testTime + ttl
			triggerChecker.ttlState = moira.TTLStateDEL
			lastCheck.Timestamp = testTime + ttl

			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)}
			lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
			checkData := newCheckData(&lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}
			oldMetricState := lastCheck.Metrics[metric]

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						Timestamp:      oldMetricState.Timestamp,
						EventTimestamp: oldMetricState.EventTimestamp,
						State:          oldMetricState.State,
						Values:         oldMetricState.Values,
						Maintenance:    oldMetricState.Maintenance,
						DeletedButKept: true,
					},
				},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateOK,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		metricState = lastCheck.Metrics[metric]
		metricState.DeletedButKept = true
		lastCheck.Metrics[metric] = metricState

		Convey("Metric on maintenance, DeletedButKept is true, ttlState is delete, but a new metric comes in and DeletedButKept becomes false", func() {
			triggerChecker.from = testTime + ttl - 5*retention
			triggerChecker.until = testTime + ttl
			triggerChecker.ttlState = moira.TTLStateDEL
			lastCheck.Timestamp = testTime + ttl + retention

			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{5}, retention, triggerChecker.from)}
			lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
			checkData := newCheckData(&lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}
			oldMetricState := lastCheck.Metrics[metric]

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics: map[string]moira.MetricState{
					metric: {
						Timestamp:      triggerChecker.from,
						EventTimestamp: oldMetricState.EventTimestamp,
						State:          oldMetricState.State,
						Values:         map[string]float64{"t1": 5},
						Maintenance:    oldMetricState.Maintenance,
						DeletedButKept: false,
					},
				},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateOK,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		metricState = lastCheck.Metrics[metric]
		metricState.Maintenance = testTime + ttl - 10*retention
		lastCheck.Metrics[metric] = metricState

		Convey("No data too long and ttlState is delete, the time for Maintenance of metric is over, so it will be deleted", func() {
			triggerChecker.from = testTime + ttl - 5*retention
			triggerChecker.until = testTime + ttl
			triggerChecker.ttlState = moira.TTLStateDEL
			lastCheck.Timestamp = testTime + ttl

			dataBase.EXPECT().RemovePatternsMetrics(triggerChecker.trigger.Patterns).Return(nil)

			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{}, retention, triggerChecker.from)}
			lastCheck.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)
			checkData := newCheckData(&lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics:                      make(map[string]moira.MetricState),
				Timestamp:                    triggerChecker.until,
				State:                        moira.StateOK,
				Score:                        0,
				LastSuccessfulCheckTimestamp: 0,
				MetricsToTargetRelation:      map[string]string{},
				Clock:                        mockTime,
			})
		})
	})

	Convey("Advanced Mode", t, func() {
		expression := "t1 + t2 > 10 ? OK : ERROR"

		triggerChecker.trigger = &moira.Trigger{
			TriggerType: "expression",
			Expression:  &expression,
			Targets:     []string{"test1", "test2"},
			Patterns:    []string{"test1", "test2"},
		}

		triggerChecker.lastCheck = &moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			State:     moira.StateNODATA,
			Timestamp: testTime - metricsTTL,
			Clock:     mockTime,
		}

		Convey("Without any metrics", func() {
			aloneMetrics := map[string]metricSource.MetricData{}
			checkData := newCheckData(triggerChecker.lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics:                 map[string]moira.MetricState{},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateNODATA,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		Convey("With empty regular metrics and the number of alone metrics does not equal the number of targets", func() {
			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{5}, retention, triggerChecker.from)}
			checkData := newCheckData(triggerChecker.lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{}

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics:                 map[string]moira.MetricState{},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateNODATA,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		Convey("With regular and alone metrics, first event", func() {
			aloneMetrics := map[string]metricSource.MetricData{"t1": *metricSource.MakeMetricData(metric, []float64{5}, retention, triggerChecker.from)}
			checkData := newCheckData(triggerChecker.lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{
				"test2": {
					"t2": *metricSource.MakeMetricData(metric, []float64{5}, retention, triggerChecker.from),
				},
			}

			mockTime.EXPECT().NowUnix().Return(testTime).Times(2)
			dataBase.EXPECT().PushNotificationEvent(
				&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: testTime + ttl - 5*retention,
					State:     moira.StateERROR,
					OldState:  moira.StateNODATA,
					Metric:    "test2",
					Values:    map[string]float64{"t1": 5, "t2": 5},
					Message:   nil,
				}, true).Return(nil)

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"test2": {
						EventTimestamp: testTime + ttl - 5*retention,
						State:          moira.StateERROR,
						Timestamp:      testTime + ttl - 5*retention,
						Values:         map[string]float64{"t1": 5, "t2": 5},
					},
				},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateNODATA,
				Score:                   0,
				Clock:                   mockTime,
			})
		})

		Convey("With only regular metrics", func() {
			aloneMetrics := map[string]metricSource.MetricData{}
			checkData := newCheckData(triggerChecker.lastCheck, triggerChecker.until)
			metricsToCheck := map[string]map[string]metricSource.MetricData{
				"test1": {
					"t1": *metricSource.MakeMetricData(metric, []float64{10}, retention, triggerChecker.from),
					"t2": *metricSource.MakeMetricData(metric, []float64{5}, retention, triggerChecker.from),
				},
			}

			mockTime.EXPECT().NowUnix().Return(testTime).Times(2)
			dataBase.EXPECT().PushNotificationEvent(
				&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: testTime + ttl - 5*retention,
					State:     moira.StateOK,
					OldState:  moira.StateNODATA,
					Metric:    "test1",
					Values:    map[string]float64{"t1": 10, "t2": 5},
					Message:   nil,
				}, true).Return(nil)

			checkData, err := triggerChecker.check(metricsToCheck, aloneMetrics, checkData, logger)
			So(err, ShouldBeNil)
			So(checkData, ShouldResemble, moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"test1": {
						EventTimestamp: testTime + ttl - 5*retention,
						State:          moira.StateOK,
						Timestamp:      testTime + ttl - 5*retention,
						Values:         map[string]float64{"t1": 10, "t2": 5},
					},
				},
				MetricsToTargetRelation: map[string]string{},
				Timestamp:               triggerChecker.until,
				State:                   moira.StateNODATA,
				Score:                   0,
				Clock:                   mockTime,
			})
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

	checkerMetrics, _ := metrics.
		ConfigureCheckerMetrics(metrics.NewDummyRegistry(), []moira.ClusterKey{defaultLocalClusterKey}).
		GetCheckMetricsBySource(defaultLocalClusterKey)
	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
		source:    source,
		logger:    logger,
		config:    &Config{},
		metrics:   checkerMetrics,
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
	dataBase.EXPECT().SetTriggerLastCheck(
		triggerChecker.triggerID,
		&lastCheck,
		triggerChecker.trigger.ClusterKey(),
	).Return(nil)
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

	checkerMetrics, _ := metrics.
		ConfigureCheckerMetrics(metrics.NewDummyRegistry(), []moira.ClusterKey{defaultLocalClusterKey}).
		GetCheckMetricsBySource(defaultLocalClusterKey)
	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
		source:    source,
		logger:    logger,
		config:    &Config{},
		metrics:   checkerMetrics,
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
	dataBase.EXPECT().SetTriggerLastCheck(
		triggerChecker.triggerID,
		&lastCheck,
		triggerChecker.trigger.ClusterKey(),
	).Return(nil).AnyTimes()

	for n := 0; n < b.N; n++ {
		err := triggerChecker.Check()
		if err != nil {
			b.Errorf("Check() returned error: %v", err)
		}
	}
}

func TestGetExpressionValues(t *testing.T) {
	logger, _ := logging.GetLogger("Test")

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

			var valueTimestamp int64 = 17
			expression, values, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeTrue)
			So(expression, ShouldResemble, expectedExpression)
			So(values, ShouldResemble, expectedValues)
		})

		Convey("last value is empty", func() {
			var valueTimestamp int64 = 67
			_, _, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("value before first value", func() {
			var valueTimestamp int64 = 11
			_, _, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("value in the middle is empty ", func() {
			var valueTimestamp int64 = 44
			_, _, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("value in the middle is valid", func() {
			expectedExpression := &expression.TriggerExpression{
				MainTargetValue:         3,
				AdditionalTargetsValues: make(map[string]float64),
			}
			expectedValues := map[string]float64{"t1": 3}

			var valueTimestamp int64 = 53
			expression, values, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
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
			var valueTimestamp int64 = 29
			_, _, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("t1 and t2 values in the middle is empty ", func() {
			var valueTimestamp int64 = 42
			_, _, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeFalse)
		})

		Convey("both first values is valid ", func() {
			expectedValues := map[string]float64{"t1": 0, "t2": 4}

			var valueTimestamp int64 = 17
			expression, values, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeTrue)
			So(expression.MainTargetValue, ShouldBeIn, []float64{0, 4})
			So(values, ShouldResemble, expectedValues)
		})
	})

	Convey("Don't evaluate the expression if we couldn't get the metric by target", t, func() {
		metricData := metricSource.MetricData{
			Name:      "test",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0.0, math.NaN(), math.NaN(), 3.0, math.NaN()},
		}
		metrics := map[string]metricSource.MetricData{
			"t2": metricData,
		}

		Convey("Couldn't get a metric by t1", func() {
			var valueTimestamp int64 = 17
			_, _, noEmptyValues := getExpressionValues(metrics, &valueTimestamp, logger)
			So(noEmptyValues, ShouldBeFalse)
		})
	})
}

func TestTriggerChecker_handlePrepareError(t *testing.T) {
	Convey("Test handlePrepareError", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger, _ := logging.GetLogger("Test")

		trigger := &moira.Trigger{
			TriggerSource: moira.GraphiteLocal,
			ClusterId:     moira.DefaultCluster,
		}
		triggerChecker := TriggerChecker{
			triggerID: "test trigger",
			trigger:   trigger,
			database:  dataBase,
			logger:    logger,
		}
		checkData := moira.CheckData{}

		Convey("with ErrTriggerHasSameMetricNames", func() {
			err := ErrTriggerHasSameMetricNames{}
			pass, checkDataReturn, errReturn := triggerChecker.handlePrepareError(checkData, err)
			So(errReturn, ShouldBeNil)
			So(pass, ShouldEqual, CanContinueCheck)
			So(checkDataReturn, ShouldResemble, moira.CheckData{
				State:   moira.StateEXCEPTION,
				Message: err.Error(),
			})
		})

		Convey("with ErrUnexpectedAloneMetric", func() {
			err := conversion.ErrUnexpectedAloneMetric{}
			checkData.Timestamp = int64(15)
			triggerChecker.lastCheck = &moira.CheckData{
				State:          moira.StateOK,
				EventTimestamp: 10,
			}
			expectedCheckData := moira.CheckData{
				Score:          100000,
				State:          moira.StateEXCEPTION,
				Message:        err.Error(),
				Timestamp:      int64(15),
				EventTimestamp: int64(15),
			}
			dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
				IsTriggerEvent:   true,
				TriggerID:        triggerChecker.triggerID,
				State:            moira.StateEXCEPTION,
				OldState:         getEventOldState(moira.StateOK, "", false),
				Timestamp:        15,
				Metric:           triggerChecker.trigger.Name,
				MessageEventInfo: nil,
			}, true)
			dataBase.EXPECT().SetTriggerLastCheck("test trigger", &expectedCheckData, trigger.ClusterKey())
			pass, checkDataReturn, errReturn := triggerChecker.handlePrepareError(checkData, err)
			So(errReturn, ShouldBeNil)
			So(pass, ShouldEqual, MustStopCheck)
			So(checkDataReturn, ShouldResemble, expectedCheckData)
		})

		Convey("with ErrEmptyAloneMetricsTarget-this error is handled as NODATA", func() {
			err := conversion.NewErrEmptyAloneMetricsTarget("t2")
			triggerChecker.lastCheck = &moira.CheckData{
				State:          moira.StateNODATA,
				EventTimestamp: 10,
			}
			expectedCheckData := moira.CheckData{
				Score:          1000,
				State:          moira.StateNODATA,
				EventTimestamp: 10,
			}
			dataBase.EXPECT().SetTriggerLastCheck("test trigger", &expectedCheckData, trigger.ClusterKey())
			pass, checkDataReturn, errReturn := triggerChecker.handlePrepareError(checkData, err)
			So(errReturn, ShouldBeNil)
			So(pass, ShouldEqual, MustStopCheck)
			So(checkDataReturn, ShouldResemble, expectedCheckData)
		})
	})
}
