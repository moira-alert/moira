package redis

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

func TestLastCheck(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	var triggerMaintenanceTS int64

	Convey("LastCheck manipulation", t, func(c C) {
		Convey("Test read write delete", t, func(c C) {
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, lastCheckTest)

			err = dataBase.RemoveTriggerLastCheck(triggerID)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldResemble, database.ErrNil)
			c.So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test no lastcheck", t, func(c C) {
			triggerID := uuid.Must(uuid.NewV4()).String()
			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldBeError)
			c.So(err, ShouldResemble, database.ErrNil)
			c.So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test set metrics check maintenance", t, func(c C) {
			Convey("While no check", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{}, nil, "", 0)
				c.So(err, ShouldBeNil)
			})

			Convey("While no metrics", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("While no metrics to change", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Has metrics to change", t, func(c C) {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, checkData)
			})
		})

		Convey("Test set Trigger and metrics check maintenance", t, func(c C) {
			Convey("While no check", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerCheckMaintenance(triggerID, make(map[string]int64), nil, "", 0)
				c.So(err, ShouldBeNil)
			})

			Convey("Set metrics maintenance while no metrics", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("Set trigger maintenance while no metrics", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				c.So(err, ShouldBeNil)

				triggerMaintenanceTS = 1000

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, &triggerMaintenanceTS, "", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, lastCheckWithNoMetricsWithMaintenance)
			})

			Convey("Set metrics maintenance while no metrics to change", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Set trigger maintenance while no metrics to change", t, func(c C) {
				newLastCheckTest := lastCheckTest
				newLastCheckTest.Maintenance = 1000
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				c.So(err, ShouldBeNil)

				triggerMaintenanceTS = 1000
				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, &triggerMaintenanceTS, "anonymous", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, newLastCheckTest)
			})

			Convey("Set metrics maintenance while has metrics to change", t, func(c C) {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, checkData)
			})

			Convey("Set trigger and metrics maintenance while has metrics to change", t, func(c C) {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				c.So(err, ShouldBeNil)

				triggerMaintenanceTS = 1000
				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, &triggerMaintenanceTS, "", 0)
				c.So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5
				checkData.Maintenance = 1000

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, checkData)
			})

			Convey("Set trigger maintenance to 0 and metrics maintenance", t, func(c C) {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				c.So(err, ShouldBeNil)

				triggerMaintenanceTS = 0
				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{}, &triggerMaintenanceTS, "", 0)
				c.So(err, ShouldBeNil)
				checkData.Maintenance = 0

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, checkData)
			})
		})

		Convey("Test last check manipulations update 'triggers to reindex' list", t, func(c C) {
			dataBase.flush()
			triggerID := uuid.Must(uuid.NewV4()).String()

			// there was no trigger with such ID, so function should return true
			c.So(dataBase.checkDataScoreChanged(triggerID, &lastCheckWithNoMetrics), ShouldBeTrue)

			// set new last check. Should add a trigger to a reindex set
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
			c.So(err, ShouldBeNil)

			c.So(dataBase.checkDataScoreChanged(triggerID, &lastCheckWithNoMetrics), ShouldBeFalse)

			c.So(dataBase.checkDataScoreChanged(triggerID, &lastCheckTest), ShouldBeTrue)

			actual, err := dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []string{triggerID})

			time.Sleep(time.Second)

			err = dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []string{triggerID})

			err = dataBase.RemoveTriggersToReindex(time.Now().Unix() + 10)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldBeEmpty)

			err = dataBase.RemoveTriggerLastCheck(triggerID)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []string{triggerID})
		})
	})
}

func TestRemoteLastCheck(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("LastCheck manipulation", t, func(c C) {
		Convey("Test read write delete", t, func(c C) {
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, true)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, lastCheckTest)

			err = dataBase.RemoveTriggerLastCheck(triggerID)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldResemble, database.ErrNil)
			c.So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test no lastcheck", t, func(c C) {
			triggerID := uuid.Must(uuid.NewV4()).String()
			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldBeError)
			c.So(err, ShouldResemble, database.ErrNil)
			c.So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test set trigger check maintenance", t, func(c C) {
			Convey("While no check", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{}, nil, "", 0)
				c.So(err, ShouldBeNil)
			})

			Convey("While no metrics", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, true)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("While no metrics to change", t, func(c C) {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, true)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Has metrics to change", t, func(c C) {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, true)
				c.So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				c.So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, checkData)
			})
		})
	})
}

func TestLastCheckErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		actual1, err := dataBase.GetTriggerLastCheck("123")
		c.So(actual1, ShouldResemble, moira.CheckData{})
		c.So(err, ShouldNotBeNil)

		err = dataBase.SetTriggerLastCheck("123", &lastCheckTest, false)
		c.So(err, ShouldNotBeNil)

		err = dataBase.RemoveTriggerLastCheck("123")
		c.So(err, ShouldNotBeNil)

		var triggerMaintenanceTS int64 = 123
		err = dataBase.SetTriggerCheckMaintenance("123", map[string]int64{}, &triggerMaintenanceTS, "", 0)
		c.So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetTriggerLastCheck("123")
		c.So(actual2, ShouldResemble, moira.CheckData{})
		c.So(err, ShouldNotBeNil)
	})
}

func TestMaintenanceUserSave(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	var triggerMaintenanceTS int64

	Convey("Check user saving for trigger maintenance", t, func(c C) {
		userLogin := "test"
		newLastCheckTest := lastCheckTest

		Convey("Start user and time", t, func(c C) {
			startTime := int64(500)
			newLastCheckTest.Maintenance = 1000
			newLastCheckTest.MaintenanceInfo.StartUser = &userLogin
			newLastCheckTest.MaintenanceInfo.StartTime = &startTime
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			c.So(err, ShouldBeNil)

			triggerMaintenanceTS = 1000
			err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, &triggerMaintenanceTS, userLogin, startTime)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, newLastCheckTest)
		})

		Convey("Stop user and time", t, func(c C) {
			startTime := int64(5000)
			newLastCheckTest.Maintenance = 1000
			newLastCheckTest.MaintenanceInfo.StopUser = &userLogin
			newLastCheckTest.MaintenanceInfo.StopTime = &startTime
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			c.So(err, ShouldBeNil)

			triggerMaintenanceTS = 1000
			err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, &triggerMaintenanceTS, userLogin, startTime)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, newLastCheckTest)
		})
	})

	Convey("Check user saving for metric maintenance", t, func(c C) {
		checkData := lastCheckTest
		triggerID := uuid.Must(uuid.NewV4()).String()
		checkData.MaintenanceInfo = moira.MaintenanceInfo{}
		userLogin := "test"
		var timeCallMaintenance = int64(3)
		err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
		c.So(err, ShouldBeNil)

		triggerMaintenanceTS = 1000
		err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, &triggerMaintenanceTS, userLogin, timeCallMaintenance)
		c.So(err, ShouldBeNil)

		metric1 := checkData.Metrics["metric1"]
		metric1.MaintenanceInfo = moira.MaintenanceInfo{}
		metric1.Maintenance = 1
		metric1.MaintenanceInfo.StopUser = &userLogin
		metric1.MaintenanceInfo.StopTime = &timeCallMaintenance

		metric5 := checkData.Metrics["metric5"]
		metric5.MaintenanceInfo = moira.MaintenanceInfo{}
		metric5.Maintenance = 5
		metric5.MaintenanceInfo.StartUser = &userLogin
		metric5.MaintenanceInfo.StartTime = &timeCallMaintenance

		checkData.Metrics["metric1"] = metric1
		checkData.Metrics["metric5"] = metric5
		checkData.Maintenance = 1000
		checkData.MaintenanceInfo.StartUser = &userLogin
		checkData.MaintenanceInfo.StartTime = &timeCallMaintenance

		actual, err := dataBase.GetTriggerLastCheck(triggerID)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldResemble, checkData)
	})
}

var lastCheckTest = moira.CheckData{
	Score:       6000,
	State:       moira.StateOK,
	Timestamp:   1504509981,
	Maintenance: 1552723340,
	Metrics: map[string]moira.MetricState{
		"metric1": {
			EventTimestamp: 1504449789,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric2": {
			EventTimestamp: 1504449789,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric3": {
			EventTimestamp: 1504449789,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric4": {
			EventTimestamp: 1504463770,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric5": {
			EventTimestamp: 1504463770,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric6": {
			EventTimestamp: 1504463770,
			State:          "Ok",
			Suppressed:     false,
			Timestamp:      1504509380,
		},
	},
}

var lastCheckWithNoMetrics = moira.CheckData{
	Score:     0,
	State:     moira.StateOK,
	Timestamp: 1504509981,
	Metrics:   make(map[string]moira.MetricState),
}

var lastCheckWithNoMetricsWithMaintenance = moira.CheckData{
	Score:       0,
	State:       moira.StateOK,
	Timestamp:   1504509981,
	Maintenance: 1000,
	Metrics:     make(map[string]moira.MetricState),
}
