package checker

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/clock"
	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

var hourInSec = int64(time.Hour.Seconds())

func TestInitTriggerChecker(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("Test")
	config := &Config{}
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := local.Create(dataBase)
	triggerID := "superId"
	checkerMetrics := metrics.ConfigureCheckerMetrics(
		metrics.NewDummyRegistry(),
		[]moira.ClusterKey{moira.DefaultLocalCluster},
	)
	defer mockCtrl.Finish()

	Convey("Test errors", t, func() {
		Convey("Get trigger error", func() {
			getTriggerError := fmt.Errorf("Oppps! Can't read trigger")
			dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{
				TriggerSource: moira.GraphiteLocal,
				ClusterId:     moira.DefaultCluster,
			}, getTriggerError)
			_, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), checkerMetrics)
			So(err, ShouldBeError)
			So(err, ShouldResemble, getTriggerError)
		})

		Convey("No trigger error", func() {
			dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{
				TriggerSource: moira.GraphiteLocal,
				ClusterId:     moira.DefaultCluster,
			}, database.ErrNil)
			_, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), checkerMetrics)
			So(err, ShouldBeError)
			So(err, ShouldResemble, ErrTriggerNotExists)
		})

		Convey("Get lastCheck error", func() {
			readLastCheckError := fmt.Errorf("Oppps! Can't read last check")
			dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{
				TriggerType:   moira.RisingTrigger,
				TriggerSource: moira.GraphiteLocal,
				ClusterId:     moira.DefaultCluster,
			}, nil)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, readLastCheckError)
			_, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), checkerMetrics)
			So(err, ShouldBeError)
			So(err, ShouldResemble, readLastCheckError)
		})
	})

	var warnValue float64 = 10000
	var errorValue float64 = 100000
	var ttl int64 = 900
	var value float64
	trigger := moira.Trigger{
		ID:            "d39b8510-b2f4-448c-b881-824658c58128",
		Name:          "Time",
		Targets:       []string{"aliasByNode(Metric.*.time, 1)"},
		WarnValue:     &warnValue,
		ErrorValue:    &errorValue,
		TriggerType:   moira.RisingTrigger,
		Tags:          []string{"tag1", "tag2"},
		TTLState:      &moira.TTLStateOK,
		Patterns:      []string{"Egais.elasticsearch.*.*.jvm.gc.collection.time"},
		TTL:           ttl,
		TriggerSource: moira.GraphiteLocal,
		ClusterId:     moira.DefaultCluster,
	}

	metrics, _ := checkerMetrics.GetCheckMetrics(&trigger)

	lastCheck := moira.CheckData{
		Timestamp: 1502694487,
		State:     moira.StateOK,
		Score:     0,
		Metrics: map[string]moira.MetricState{
			"1": {
				Timestamp:      1502694427,
				State:          moira.StateOK,
				Suppressed:     false,
				Values:         map[string]float64{"t1": value}, //nolint
				EventTimestamp: 1501680428,
			},
			"2": {
				Timestamp:      1502694427,
				State:          moira.StateOK,
				Suppressed:     false,
				Values:         map[string]float64{"t1": value}, //nolint
				EventTimestamp: 1501679827,
			},
			"3": {
				Timestamp:      1502694427,
				State:          moira.StateOK,
				Suppressed:     false,
				Values:         map[string]float64{"t1": value}, //nolint
				EventTimestamp: 1501679887,
			},
		},
	}

	Convey("Test trigger checker with lastCheck", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), checkerMetrics)
		So(err, ShouldBeNil)

		expectedLastCheck := lastCheck
		expectedLastCheck.Clock = clock.NewSystemClock()
		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    actual.logger,
			trigger:   &trigger,
			ttl:       trigger.TTL,
			ttlState:  *trigger.TTLState,
			lastCheck: &expectedLastCheck,
			from:      lastCheck.Timestamp - ttl,
			until:     actual.until,
			metrics:   metrics,
		}
		So(actual, ShouldResemble, &expected)
	})

	Convey("Test trigger checker without lastCheck", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), checkerMetrics)
		So(err, ShouldBeNil)

		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    actual.logger,
			trigger:   &trigger,
			ttl:       trigger.TTL,
			ttlState:  *trigger.TTLState,
			lastCheck: &moira.CheckData{
				Metrics:   make(map[string]moira.MetricState),
				State:     moira.StateOK,
				Timestamp: actual.until - hourInSec,
				Clock:     clock.NewSystemClock(),
			},
			from:    actual.until - hourInSec - ttl,
			until:   actual.until,
			metrics: metrics,
		}

		So(*actual, ShouldResemble, expected)
	})

	trigger.TTL = 0
	trigger.TTLState = nil

	Convey("Test trigger checker without lastCheck and ttl", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), checkerMetrics)
		So(err, ShouldBeNil)

		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    actual.logger,
			trigger:   &trigger,
			ttl:       0,
			ttlState:  moira.TTLStateNODATA,
			lastCheck: &moira.CheckData{
				Metrics:   make(map[string]moira.MetricState),
				State:     moira.StateOK,
				Timestamp: actual.until - hourInSec,
				Clock:     clock.NewSystemClock(),
			},
			from:    actual.until - hourInSec - tenMinInSec,
			until:   actual.until,
			metrics: metrics,
		}
		So(*actual, ShouldResemble, expected)
	})

	Convey("Test trigger checker with lastCheck and without ttl", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), checkerMetrics)

		So(err, ShouldBeNil)

		expectedLastCheck := lastCheck
		expectedLastCheck.Clock = clock.NewSystemClock()
		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    actual.logger,
			trigger:   &trigger,
			ttl:       0,
			ttlState:  moira.TTLStateNODATA,
			lastCheck: &expectedLastCheck,
			from:      lastCheck.Timestamp - tenMinInSec,
			until:     actual.until,
			metrics:   metrics,
		}
		So(*actual, ShouldResemble, expected)
	})
}
