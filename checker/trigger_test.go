package checker

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInitTriggerChecker(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	triggerChecker := TriggerChecker{
		TriggerID: "superId",
		Database:  dataBase,
		Logger:    logger,
	}

	Convey("Test errors", t, func() {
		Convey("Get trigger error", func() {
			getTriggerError := fmt.Errorf("Oppps! Can't read trigger")
			dataBase.EXPECT().GetTrigger(triggerChecker.TriggerID).Return(moira.Trigger{}, getTriggerError)
			err := triggerChecker.InitTriggerChecker()
			So(err, ShouldBeError)
			So(err, ShouldResemble, getTriggerError)
		})

		Convey("No trigger error", func() {
			dataBase.EXPECT().GetTrigger(triggerChecker.TriggerID).Return(moira.Trigger{}, database.ErrNil)
			err := triggerChecker.InitTriggerChecker()
			So(err, ShouldBeError)
			So(err, ShouldResemble, ErrTriggerNotExists)
		})

		Convey("Get lastCheck error", func() {
			readLastCheckError := fmt.Errorf("Oppps! Can't read last check")
			dataBase.EXPECT().GetTrigger(triggerChecker.TriggerID).Return(moira.Trigger{}, nil)
			dataBase.EXPECT().GetTriggerLastCheck(triggerChecker.TriggerID).Return(nil, readLastCheckError)
			err := triggerChecker.InitTriggerChecker()
			So(err, ShouldBeError)
			So(err, ShouldResemble, readLastCheckError)
		})
	})

	var warnWalue float64 = 10000
	var errorWalue float64 = 10000
	var ttl int64 = 900
	var value float64
	ttlStateOk := OK
	ttlStateNoData := NODATA

	trigger := moira.Trigger{
		ID:              "d39b8510-b2f4-448c-b881-824658c58128",
		Name:            "Time",
		Targets:         []string{"aliasByNode(Metric.*.time, 1)"},
		WarnValue:       &warnWalue,
		ErrorValue:      &errorWalue,
		Tags:            []string{"tag1", "tag2"},
		TTLState:        &ttlStateOk,
		Patterns:        []string{"Egais.elasticsearch.*.*.jvm.gc.collection.time"},
		TTL:             &ttl,
		IsSimpleTrigger: false,
	}

	lastCheck := moira.CheckData{
		Timestamp: 1502694487,
		State:     OK,
		Score:     0,
		Metrics: map[string]moira.MetricState{
			"1": {
				Timestamp:      1502694427,
				State:          OK,
				Suppressed:     false,
				Value:          &value,
				EventTimestamp: 1501680428,
			},
			"2": {
				Timestamp:      1502694427,
				State:          OK,
				Suppressed:     false,
				Value:          &value,
				EventTimestamp: 1501679827,
			},
			"3": {
				Timestamp:      1502694427,
				State:          OK,
				Suppressed:     false,
				Value:          &value,
				EventTimestamp: 1501679887,
			},
		},
	}

	triggerChecker = TriggerChecker{
		TriggerID: trigger.ID,
		Database:  dataBase,
		Logger:    logger,
	}

	Convey("Test trigger checker with lastCheck", t, func() {
		dataBase.EXPECT().GetTrigger(triggerChecker.TriggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerChecker.TriggerID).Return(&lastCheck, nil)
		err := triggerChecker.InitTriggerChecker()
		So(err, ShouldBeNil)

		expectedTriggerChecker := triggerChecker
		expectedTriggerChecker.trigger = &trigger
		expectedTriggerChecker.isSimple = trigger.IsSimpleTrigger
		expectedTriggerChecker.ttl = trigger.TTL
		expectedTriggerChecker.ttlState = *trigger.TTLState
		expectedTriggerChecker.lastCheck = &lastCheck
		expectedTriggerChecker.From = lastCheck.Timestamp - ttl
		So(triggerChecker, ShouldResemble, expectedTriggerChecker)
	})

	Convey("Test trigger checker without lastCheck", t, func() {
		dataBase.EXPECT().GetTrigger(triggerChecker.TriggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerChecker.TriggerID).Return(nil, nil)
		err := triggerChecker.InitTriggerChecker()
		So(err, ShouldBeNil)

		expectedTriggerChecker := triggerChecker
		expectedTriggerChecker.trigger = &trigger
		expectedTriggerChecker.isSimple = trigger.IsSimpleTrigger
		expectedTriggerChecker.ttl = trigger.TTL
		expectedTriggerChecker.ttlState = *trigger.TTLState
		expectedTriggerChecker.lastCheck = &moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			State:     NODATA,
			Timestamp: expectedTriggerChecker.Until - 3600,
		}
		expectedTriggerChecker.From = expectedTriggerChecker.Until - 3600 - ttl
		So(triggerChecker, ShouldResemble, expectedTriggerChecker)
	})

	trigger.TTL = nil
	trigger.TTLState = nil

	Convey("Test trigger checker without lastCheck and ttl", t, func() {
		dataBase.EXPECT().GetTrigger(triggerChecker.TriggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerChecker.TriggerID).Return(nil, nil)
		err := triggerChecker.InitTriggerChecker()
		So(err, ShouldBeNil)

		expectedTriggerChecker := triggerChecker
		expectedTriggerChecker.trigger = &trigger
		expectedTriggerChecker.isSimple = trigger.IsSimpleTrigger
		expectedTriggerChecker.ttl = nil
		expectedTriggerChecker.ttlState = ttlStateNoData
		expectedTriggerChecker.lastCheck = &moira.CheckData{
			Metrics:   make(map[string]moira.MetricState),
			State:     NODATA,
			Timestamp: expectedTriggerChecker.Until - 3600,
		}
		expectedTriggerChecker.From = expectedTriggerChecker.Until - 3600 - 600
		So(triggerChecker, ShouldResemble, expectedTriggerChecker)
	})

	Convey("Test trigger checker with lastCheck and without ttl", t, func() {
		dataBase.EXPECT().GetTrigger(triggerChecker.TriggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerLastCheck(triggerChecker.TriggerID).Return(&lastCheck, nil)
		err := triggerChecker.InitTriggerChecker()
		So(err, ShouldBeNil)

		expectedTriggerChecker := triggerChecker
		expectedTriggerChecker.trigger = &trigger
		expectedTriggerChecker.isSimple = trigger.IsSimpleTrigger
		expectedTriggerChecker.ttl = nil
		expectedTriggerChecker.ttlState = ttlStateNoData
		expectedTriggerChecker.lastCheck = &lastCheck
		expectedTriggerChecker.From = lastCheck.Timestamp - 600
		So(triggerChecker, ShouldResemble, expectedTriggerChecker)
	})
}
