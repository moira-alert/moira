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

	Convey("LastCheck manipulation", t, func() {
		Convey("Test read write delete", func() {
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, lastCheckTest)

			err = dataBase.RemoveTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)

			actual, err = dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test no lastcheck", func() {
			triggerID := uuid.Must(uuid.NewV4()).String()
			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeError)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test set metrics check maintenance", func() {
			Convey("While no check", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{}, nil, "", 0)
				So(err, ShouldBeNil)
			})

			Convey("While no metrics", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("While no metrics to change", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, nil, "", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Has metrics to change", func() {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, checkData)
			})
		})

		Convey("Test set Trigger and metrics check maintenance", func() {
			Convey("While no check", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerCheckMaintenance(triggerID, make(map[string]int64), nil, "", 0)
				So(err, ShouldBeNil)
			})

			Convey("Set metrics maintenance while no metrics", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("Set trigger maintenance while no metrics", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				So(err, ShouldBeNil)

				triggerMaintenanceTS = 1000

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, &triggerMaintenanceTS, "", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetricsWithMaintenance)
			})

			Convey("Set metrics maintenance while no metrics to change", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, nil, "", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Set trigger maintenance while no metrics to change", func() {
				newLastCheckTest := lastCheckTest
				newLastCheckTest.Maintenance = 1000
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				So(err, ShouldBeNil)

				triggerMaintenanceTS = 1000
				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, &triggerMaintenanceTS, "anonymous", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, newLastCheckTest)
			})

			Convey("Set metrics maintenance while has metrics to change", func() {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, checkData)
			})

			Convey("Set trigger and metrics maintenance while has metrics to change", func() {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				So(err, ShouldBeNil)

				triggerMaintenanceTS = 1000
				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, &triggerMaintenanceTS, "", 0)
				So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5
				checkData.Maintenance = 1000

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, checkData)
			})

			Convey("Set trigger maintenance to 0 and metrics maintenance", func() {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				So(err, ShouldBeNil)

				triggerMaintenanceTS = 0
				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{}, &triggerMaintenanceTS, "", 0)
				So(err, ShouldBeNil)
				checkData.Maintenance = 0

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, checkData)
			})
		})

		Convey("Test last check manipulations update 'triggers to reindex' list", func() {
			dataBase.flush()
			triggerID := uuid.Must(uuid.NewV4()).String()

			// there was no trigger with such ID, so function should return true
			So(dataBase.checkDataScoreChanged(triggerID, &lastCheckWithNoMetrics), ShouldBeTrue)

			// set new last check. Should add a trigger to a reindex set
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
			So(err, ShouldBeNil)

			So(dataBase.checkDataScoreChanged(triggerID, &lastCheckWithNoMetrics), ShouldBeFalse)

			So(dataBase.checkDataScoreChanged(triggerID, &lastCheckTest), ShouldBeTrue)

			actual, err := dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{triggerID})

			time.Sleep(time.Second)

			err = dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{triggerID})

			err = dataBase.RemoveTriggersToReindex(time.Now().Unix() + 10)
			So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			So(err, ShouldBeNil)
			So(actual, ShouldBeEmpty)

			err = dataBase.RemoveTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{triggerID})
		})

		Convey("Test populate metric values", func() {
			value := float64(1)
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &moira.CheckData{
				Score:       6000,
				State:       moira.StateOK,
				Timestamp:   1504509981,
				Maintenance: 1552723340,
				Metrics: map[string]moira.MetricState{
					"metric1": {
						EventTimestamp: 1504463770,
						State:          "Ok",
						Suppressed:     false,
						Timestamp:      1504509380,
						Value:          &value,
					},
				},
			}, false)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, moira.CheckData{
				Score:       6000,
				State:       moira.StateOK,
				Timestamp:   1504509981,
				Maintenance: 1552723340,
				Metrics: map[string]moira.MetricState{
					"metric1": {
						EventTimestamp: 1504463770,
						State:          "Ok",
						Suppressed:     false,
						Timestamp:      1504509380,
						Values:         map[string]float64{"t1": 1},
					},
				},
				MetricsToTargetRelation: map[string]string{},
			})
		})
	})
}

func TestRemoteLastCheck(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("LastCheck manipulation", t, func() {
		Convey("Test read write delete", func() {
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, true)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, lastCheckTest)

			err = dataBase.RemoveTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)

			actual, err = dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test no lastcheck", func() {
			triggerID := uuid.Must(uuid.NewV4()).String()
			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeError)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test set trigger check maintenance", func() {
			Convey("While no check", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{}, nil, "", 0)
				So(err, ShouldBeNil)
			})

			Convey("While no metrics", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, true)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("While no metrics to change", func() {
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, true)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, nil, "", 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Has metrics to change", func() {
				checkData := lastCheckTest
				triggerID := uuid.Must(uuid.NewV4()).String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, true)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, nil, "", 0)
				So(err, ShouldBeNil)
				metric1 := checkData.Metrics["metric1"]
				metric5 := checkData.Metrics["metric5"]
				metric1.Maintenance = 1
				metric5.Maintenance = 5
				checkData.Metrics["metric1"] = metric1
				checkData.Metrics["metric5"] = metric5

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, checkData)
			})
		})
	})
}

func TestLastCheckErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		actual1, err := dataBase.GetTriggerLastCheck("123")
		So(actual1, ShouldResemble, moira.CheckData{})
		So(err, ShouldNotBeNil)

		err = dataBase.SetTriggerLastCheck("123", &lastCheckTest, false)
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveTriggerLastCheck("123")
		So(err, ShouldNotBeNil)

		var triggerMaintenanceTS int64 = 123
		err = dataBase.SetTriggerCheckMaintenance("123", map[string]int64{}, &triggerMaintenanceTS, "", 0)
		So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetTriggerLastCheck("123")
		So(actual2, ShouldResemble, moira.CheckData{})
		So(err, ShouldNotBeNil)
	})
}

func TestMaintenanceUserSave(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	var triggerMaintenanceTS int64

	Convey("Check user saving for trigger maintenance", t, func() {
		userLogin := "test"
		newLastCheckTest := lastCheckTest

		Convey("Start user and time", func() {
			startTime := int64(500)
			newLastCheckTest.Maintenance = 1000
			newLastCheckTest.MaintenanceInfo.StartUser = &userLogin
			newLastCheckTest.MaintenanceInfo.StartTime = &startTime
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			So(err, ShouldBeNil)

			triggerMaintenanceTS = 1000
			err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, &triggerMaintenanceTS, userLogin, startTime)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, newLastCheckTest)
		})

		Convey("Stop user and time", func() {
			startTime := int64(5000)
			newLastCheckTest.Maintenance = 1000
			newLastCheckTest.MaintenanceInfo.StopUser = &userLogin
			newLastCheckTest.MaintenanceInfo.StopTime = &startTime
			triggerID := uuid.Must(uuid.NewV4()).String()
			err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
			So(err, ShouldBeNil)

			triggerMaintenanceTS = 1000
			err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, &triggerMaintenanceTS, userLogin, startTime)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, newLastCheckTest)
		})
	})

	Convey("Check user saving for metric maintenance", t, func() {
		checkData := lastCheckTest
		triggerID := uuid.Must(uuid.NewV4()).String()
		checkData.MaintenanceInfo = moira.MaintenanceInfo{}
		userLogin := "test"
		var timeCallMaintenance = int64(3)
		err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
		So(err, ShouldBeNil)

		triggerMaintenanceTS = 1000
		err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, &triggerMaintenanceTS, userLogin, timeCallMaintenance)
		So(err, ShouldBeNil)

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
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, checkData)
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
			Values:         map[string]float64{},
		},
		"metric2": {
			EventTimestamp: 1504449789,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
			Values:         map[string]float64{},
		},
		"metric3": {
			EventTimestamp: 1504449789,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
			Values:         map[string]float64{},
		},
		"metric4": {
			EventTimestamp: 1504463770,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
			Values:         map[string]float64{},
		},
		"metric5": {
			EventTimestamp: 1504463770,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
			Values:         map[string]float64{},
		},
		"metric6": {
			EventTimestamp: 1504463770,
			State:          "Ok",
			Suppressed:     false,
			Timestamp:      1504509380,
			Values:         map[string]float64{},
		},
		"metric7": {
			EventTimestamp: 1504463770,
			State:          "Ok",
			Suppressed:     false,
			Timestamp:      1504509380,
			Values:         map[string]float64{},
		},
	},
	MetricsToTargetRelation: map[string]string{},
}

var lastCheckWithNoMetrics = moira.CheckData{
	Score:                   0,
	State:                   moira.StateOK,
	Timestamp:               1504509981,
	Metrics:                 make(map[string]moira.MetricState),
	MetricsToTargetRelation: map[string]string{},
}

var lastCheckWithNoMetricsWithMaintenance = moira.CheckData{
	Score:                   0,
	State:                   moira.StateOK,
	Timestamp:               1504509981,
	Maintenance:             1000,
	Metrics:                 make(map[string]moira.MetricState),
	MetricsToTargetRelation: map[string]string{},
}
