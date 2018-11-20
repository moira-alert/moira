package redis

import (
	"testing"

	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

func TestLastCheck(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("LastCheck manipulation", t, func() {
		Convey("Test read write delete", func() {
			triggerID := uuid.NewV4().String()
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
			triggerID := uuid.NewV4().String()
			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeError)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test set metrics check maintenance", func() {
			Convey("While no check", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{})
				So(err, ShouldBeNil)
			})

			Convey("While no metrics", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5})
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("While no metrics to change", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5})
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Has metrics to change", func() {
				checkData := lastCheckTest
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5})
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
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerCheckMaintenance(triggerID, make(map[string]int64), 0)
				So(err, ShouldBeNil)
			})

			Convey("Set metrics maintenance while no metrics", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("Set trigger maintenance while no metrics", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, 1000)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetricsWithMaintenance)
			})

			Convey("Set metrics maintenance while no metrics to change", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, 0)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Set trigger maintenance while no metrics to change", func() {
				newLastCheckTest := lastCheckTest
				newLastCheckTest.Maintenance = 1000
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5}, 1000)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, newLastCheckTest)
			})

			Convey("Set metrics maintenance while has metrics to change", func() {
				checkData := lastCheckTest
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, 0)
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
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, false)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, 1000)
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
		})

		Convey("Test get trigger check ids", func() {
			dataBase.flush()
			okTriggerID := uuid.NewV4().String()
			badTriggerID := uuid.NewV4().String()
			err := dataBase.SetTriggerLastCheck(okTriggerID, &lastCheckWithNoMetrics, false)
			So(err, ShouldBeNil)
			err = dataBase.SetTriggerLastCheck(badTriggerID, &lastCheckTest, false)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerCheckIDs(make([]string, 0), true)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{badTriggerID})

			actual, err = dataBase.GetTriggerCheckIDs(make([]string, 0), false)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{badTriggerID, okTriggerID})
		})
	})
}

func TestRemoteLastCheck(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("LastCheck manipulation", t, func() {
		Convey("Test read write delete", func() {
			triggerID := uuid.NewV4().String()
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
			triggerID := uuid.NewV4().String()
			actual, err := dataBase.GetTriggerLastCheck(triggerID)
			So(err, ShouldBeError)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.CheckData{})
		})

		Convey("Test set trigger check maintenance", func() {
			Convey("While no check", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{})
				So(err, ShouldBeNil)
			})

			Convey("While no metrics", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckWithNoMetrics, true)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5})
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckWithNoMetrics)
			})

			Convey("While no metrics to change", func() {
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &lastCheckTest, true)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{"metric11": 1, "metric55": 5})
				So(err, ShouldBeNil)

				actual, err := dataBase.GetTriggerLastCheck(triggerID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, lastCheckTest)
			})

			Convey("Has metrics to change", func() {
				checkData := lastCheckTest
				triggerID := uuid.NewV4().String()
				err := dataBase.SetTriggerLastCheck(triggerID, &checkData, true)
				So(err, ShouldBeNil)

				err = dataBase.SetTriggerCheckMetricsMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5})
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

		Convey("Test get trigger check ids", func() {
			dataBase.flush()
			okTriggerID := uuid.NewV4().String()
			badTriggerID := uuid.NewV4().String()
			err := dataBase.SetTriggerLastCheck(okTriggerID, &lastCheckWithNoMetrics, true)
			So(err, ShouldBeNil)
			err = dataBase.SetTriggerLastCheck(badTriggerID, &lastCheckTest, true)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTriggerCheckIDs(make([]string, 0), true)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{badTriggerID})

			actual, err = dataBase.GetTriggerCheckIDs(make([]string, 0), false)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{badTriggerID, okTriggerID})
		})
	})
}

func TestLastCheckErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, emptyConfig)
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

		err = dataBase.SetTriggerCheckMetricsMaintenance("123", map[string]int64{})
		So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetTriggerCheckIDs(make([]string, 0), true)
		So(actual2, ShouldResemble, []string(nil))
		So(err, ShouldNotBeNil)
	})
}

var lastCheckTest = moira.CheckData{
	Score:     6000,
	State:     "OK",
	Timestamp: 1504509981,
	Metrics: map[string]moira.MetricState{
		"metric1": {
			EventTimestamp: 1504449789,
			State:          "NODATA",
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric2": {
			EventTimestamp: 1504449789,
			State:          "NODATA",
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric3": {
			EventTimestamp: 1504449789,
			State:          "NODATA",
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric4": {
			EventTimestamp: 1504463770,
			State:          "NODATA",
			Suppressed:     false,
			Timestamp:      1504509380,
		},
		"metric5": {
			EventTimestamp: 1504463770,
			State:          "NODATA",
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
	State:     "OK",
	Timestamp: 1504509981,
	Metrics:   make(map[string]moira.MetricState),
}

var lastCheckWithNoMetricsWithMaintenance = moira.CheckData{
	Score:       0,
	State:       "OK",
	Timestamp:   1504509981,
	Maintenance: 1000,
	Metrics:     make(map[string]moira.MetricState),
}
