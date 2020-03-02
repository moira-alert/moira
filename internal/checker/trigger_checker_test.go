package checker

import (
	"fmt"
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/internal/database"
	metricSource "github.com/moira-alert/moira/internal/metric_source"
	"github.com/moira-alert/moira/internal/metric_source/local"
	"github.com/moira-alert/moira/internal/metrics"
	mock_moira_alert "github.com/moira-alert/moira/internal/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInitTriggerChecker(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("Test")
	config := &Config{}
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := local.Create(dataBase)
	triggerID := "superId"
	defer mockCtrl.Finish()

	Convey("Test errors", t, func() {
		Convey("Get trigger error", func() {
			getTriggerError := fmt.Errorf("Oppps! Can't read trigger")
			dataBase.EXPECT().GetTrigger(triggerID).Return(moira2.Trigger{}, getTriggerError)
			_, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
			So(err, ShouldBeError)
			So(err, ShouldResemble, getTriggerError)
		})

		Convey("No trigger error", func() {
			dataBase.EXPECT().GetTrigger(triggerID).Return(moira2.Trigger{}, database.ErrNil)
			_, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
			So(err, ShouldBeError)
			So(err, ShouldResemble, ErrTriggerNotExists)
		})

		Convey("Get lastCheck error", func() {
			readLastCheckError := fmt.Errorf("Oppps! Can't read last check")
			dataBase.EXPECT().GetTrigger(triggerID).Return(moira2.Trigger{TriggerType: moira2.RisingTrigger}, nil)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira2.CheckData{}, readLastCheckError)
			_, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
			So(err, ShouldBeError)
			So(err, ShouldResemble, readLastCheckError)
		})
	})

	var warnValue float64 = 10000
	var errorValue float64 = 100000
	var ttl int64 = 900
	var value float64

	trigger := moira2.Trigger{
		ID:          "d39b8510-b2f4-448c-b881-824658c58128",
		Name:        "Time",
		Targets:     []string{"aliasByNode(Metric.*.time, 1)"},
		WarnValue:   &warnValue,
		ErrorValue:  &errorValue,
		TriggerType: moira2.RisingTrigger,
		Tags:        []string{"tag1", "tag2"},
		TTLState:    &moira2.TTLStateOK,
		Patterns:    []string{"Egais.elasticsearch.*.*.jvm.gc.collection.time"},
		TTL:         ttl,
	}

	lastCheck := moira2.CheckData{
		Timestamp: 1502694487,
		State:     moira2.StateOK,
		Score:     0,
		Metrics: map[string]moira2.MetricState{
			"1": {
				Timestamp:      1502694427,
				State:          moira2.StateOK,
				Suppressed:     false,
				Value:          &value,
				EventTimestamp: 1501680428,
			},
			"2": {
				Timestamp:      1502694427,
				State:          moira2.StateOK,
				Suppressed:     false,
				Value:          &value,
				EventTimestamp: 1501679827,
			},
			"3": {
				Timestamp:      1502694427,
				State:          moira2.StateOK,
				Suppressed:     false,
				Value:          &value,
				EventTimestamp: 1501679887,
			},
		},
	}

	Convey("Test trigger checker with lastCheck", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
		So(err, ShouldBeNil)

		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    logger,
			trigger:   &trigger,
			ttl:       trigger.TTL,
			ttlState:  *trigger.TTLState,
			lastCheck: &lastCheck,
			from:      lastCheck.Timestamp - ttl,
			until:     actual.until,
		}
		So(*actual, ShouldResemble, expected)
	})

	Convey("Test trigger checker without lastCheck", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira2.CheckData{}, database.ErrNil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
		So(err, ShouldBeNil)

		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    logger,
			trigger:   &trigger,
			ttl:       trigger.TTL,
			ttlState:  *trigger.TTLState,
			lastCheck: &moira2.CheckData{
				Metrics:   make(map[string]moira2.MetricState),
				State:     moira2.StateOK,
				Timestamp: actual.until - 3600,
			},
			from:  actual.until - 3600 - ttl,
			until: actual.until,
		}
		So(*actual, ShouldResemble, expected)
	})

	trigger.TTL = 0
	trigger.TTLState = nil

	Convey("Test trigger checker without lastCheck and ttl", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira2.CheckData{}, database.ErrNil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
		So(err, ShouldBeNil)

		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    logger,
			trigger:   &trigger,
			ttl:       0,
			ttlState:  moira2.TTLStateNODATA,
			lastCheck: &moira2.CheckData{
				Metrics:   make(map[string]moira2.MetricState),
				State:     moira2.StateOK,
				Timestamp: actual.until - 3600,
			},
			from:  actual.until - 3600 - 600,
			until: actual.until,
		}
		So(*actual, ShouldResemble, expected)
	})

	Convey("Test trigger checker with lastCheck and without ttl", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		actual, err := MakeTriggerChecker(triggerID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
		So(err, ShouldBeNil)

		expected := TriggerChecker{
			triggerID: triggerID,
			database:  dataBase,
			config:    config,
			source:    localSource,
			logger:    logger,
			trigger:   &trigger,
			ttl:       0,
			ttlState:  moira2.TTLStateNODATA,
			lastCheck: &lastCheck,
			from:      lastCheck.Timestamp - 600,
			until:     actual.until,
		}
		So(*actual, ShouldResemble, expected)
	})
}
