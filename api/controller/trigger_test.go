package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUpdateTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success update", t, func() {
		triggerModel := dto.TriggerModel{ID: uuid.Must(uuid.NewV4()).String()}
		trigger := triggerModel.ToMoiraTrigger()
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(*trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), trigger.TriggerSource).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), trigger).Return(nil)
		resp, err := UpdateTrigger(dataBase, &triggerModel, triggerModel.ID, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger updated")
	})

	Convey("Trigger does not exists", t, func() {
		trigger := dto.TriggerModel{ID: uuid.Must(uuid.NewV4()).String()}
		dataBase.EXPECT().GetTrigger(trigger.ID).Return(moira.Trigger{}, database.ErrNil)
		resp, err := UpdateTrigger(dataBase, &trigger, trigger.ID, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("trigger with ID = '%s' does not exists", trigger.ID)))
		So(resp, ShouldBeNil)
	})

	Convey("Get trigger error", t, func() {
		trigger := dto.TriggerModel{ID: uuid.Must(uuid.NewV4()).String()}
		expected := fmt.Errorf("soo bad trigger")
		dataBase.EXPECT().GetTrigger(trigger.ID).Return(moira.Trigger{}, expected)
		resp, err := UpdateTrigger(dataBase, &trigger, trigger.ID, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})
}

func TestSaveTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	trigger := moira.Trigger{ID: triggerID}
	lastCheck := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {},
			"super.metric2": {},
		},
		MetricsToTargetRelation: map[string]string{
			"t2": "super.metric3",
		},
	}
	emptyLastCheck := moira.CheckData{
		Metrics:                 make(map[string]moira.MetricState),
		MetricsToTargetRelation: map[string]string{},
	}

	Convey("No timeSeries", t, func() {
		Convey("No last check", func() {
			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any(), trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})
		Convey("Has last check", func() {
			actualLastCheck := lastCheck
			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(actualLastCheck, nil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &emptyLastCheck, trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})
	})

	Convey("Has timeSeries", t, func() {
		actualLastCheck := lastCheck
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any(), trigger.TriggerSource).Return(nil)
		dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
		resp, err := saveTrigger(dataBase, &trigger, triggerID, map[string]bool{"super.metric1": true, "super.metric2": true})
		So(err, ShouldBeNil)
		So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		So(actualLastCheck, ShouldResemble, lastCheck)
	})

	Convey("Errors", t, func() {
		Convey("AcquireTriggerCheckLock error", func() {
			expected := fmt.Errorf("acquireTriggerCheckLock error")
			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(expected)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("GetTriggerLastCheck error", func() {
			expected := fmt.Errorf("getTriggerLastCheck error")
			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, expected)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("SetTriggerLastCheck error", func() {
			expected := fmt.Errorf("setTriggerLastCheck error")
			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any(), trigger.TriggerSource).Return(expected)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("saveTrigger error", func() {
			expected := fmt.Errorf("saveTrigger error")
			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any(), trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(expected)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})
	})
}

func TestVariousTtlState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerID := uuid.Must(uuid.NewV4()).String()
	trigger := moira.Trigger{ID: triggerID, TTLState: nil}
	lastCheck := moira.CheckData{
		Metrics: make(map[string]moira.MetricState),
		State:   moira.StateNODATA,
		Score:   1000,
	}

	Convey("Various TTLState", t, func() {
		Convey("NODATA TTLState", func() {
			trigger.TTLState = &moira.TTLStateNODATA
			lastCheck.State = moira.StateNODATA
			lastCheck.Score = 1000

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("ERROR TTLState", func() {
			trigger.TTLState = &moira.TTLStateERROR
			lastCheck.State = moira.StateERROR
			lastCheck.Score = 100

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("WARN TTLState", func() {
			trigger.TTLState = &moira.TTLStateWARN
			lastCheck.State = moira.StateWARN
			lastCheck.Score = 1

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("OK TTLState", func() {
			trigger.TTLState = &moira.TTLStateOK
			lastCheck.State = moira.StateOK
			lastCheck.Score = 0

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("DEL TTLState", func() {
			trigger.TTLState = &moira.TTLStateDEL
			lastCheck.State = moira.StateOK
			lastCheck.Score = 0

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.TriggerSource).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})
	})
}

func TestGetTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	triggerModel := dto.TriggerModel{ID: triggerID}
	trigger := *(triggerModel.ToMoiraTrigger())
	beginning := time.Unix(0, 0)
	now := time.Now()
	tomorrow := now.Add(time.Hour * 24)
	yesterday := now.Add(-time.Hour * 24)

	Convey("Has trigger no throttling", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerThrottling(triggerID).Return(beginning, beginning)
		actual, err := GetTrigger(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.Trigger{TriggerModel: triggerModel, Throttling: 0})
	})

	Convey("Has trigger has throttling", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerThrottling(triggerID).Return(tomorrow, beginning)
		actual, err := GetTrigger(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.Trigger{TriggerModel: triggerModel, Throttling: tomorrow.Unix()})
	})

	Convey("Has trigger has old throttling", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerThrottling(triggerID).Return(yesterday, beginning)
		actual, err := GetTrigger(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.Trigger{TriggerModel: triggerModel, Throttling: 0})
	})

	Convey("GetTrigger error", t, func() {
		expected := fmt.Errorf("getTrigger error")
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, expected)
		actual, err := GetTrigger(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(actual, ShouldBeNil)
	})

	Convey("No trigger", t, func() {
		dataBase.EXPECT().GetTrigger(triggerID).Return(moira.Trigger{}, database.ErrNil)
		actual, err := GetTrigger(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorNotFound("trigger not found"))
		So(actual, ShouldBeNil)
	})
}

func TestRemoveTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func() {
		dataBase.EXPECT().RemoveTrigger(triggerID).Return(nil)
		err := RemoveTrigger(dataBase, triggerID)
		So(err, ShouldBeNil)
	})

	Convey("Error remove trigger", t, func() {
		expected := fmt.Errorf("oooops! Error delete")
		dataBase.EXPECT().RemoveTrigger(triggerID).Return(expected)
		err := RemoveTrigger(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("Error remove last check", t, func() {
		expected := fmt.Errorf("oooops! Error delete")
		dataBase.EXPECT().RemoveTrigger(triggerID).Return(expected)
		err := RemoveTrigger(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestGetTriggerThrottling(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	begging := time.Unix(0, 0)
	now := time.Now()
	tomorrow := now.Add(time.Hour * 24)
	yesterday := now.Add(-time.Hour * 24)

	Convey("no throttling", t, func() {
		dataBase.EXPECT().GetTriggerThrottling(triggerID).Return(begging, begging)
		actual, err := GetTriggerThrottling(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ThrottlingResponse{Throttling: 0})
	})

	Convey("has throttling", t, func() {
		dataBase.EXPECT().GetTriggerThrottling(triggerID).Return(tomorrow, begging)
		actual, err := GetTriggerThrottling(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ThrottlingResponse{Throttling: tomorrow.Unix()})
	})

	Convey("has old throttling", t, func() {
		dataBase.EXPECT().GetTriggerThrottling(triggerID).Return(yesterday, begging)
		actual, err := GetTriggerThrottling(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ThrottlingResponse{Throttling: 0})
	})
}

func TestGetTriggerLastCheck(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	lastCheck := moira.CheckData{}

	Convey("Success", t, func() {
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		check, err := GetTriggerLastCheck(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(check, ShouldResemble, &dto.TriggerCheck{
			TriggerID: triggerID,
			CheckData: &lastCheck,
		})
	})

	Convey("Returns all metrics, because their NeedToDeleteAfterMaintenance is false", t, func() {
		lastCheck = moira.CheckData{
			Metrics: map[string]moira.MetricState{
				"metric":  {},
				"metric2": {},
			},
		}
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		check, err := GetTriggerLastCheck(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(check, ShouldResemble, &dto.TriggerCheck{
			TriggerID: triggerID,
			CheckData: &lastCheck,
		})
	})

	Convey("Does not return all metrics, as some NeedToDeleteAfterMaintenance is true", t, func() {
		lastCheck = moira.CheckData{
			Metrics: map[string]moira.MetricState{
				"metric": {
					NeedToDeleteAfterMaintenance: true,
				},
				"metric2": {},
			},
		}
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(lastCheck, nil)
		check, err := GetTriggerLastCheck(dataBase, triggerID)
		So(err, ShouldBeNil)
		So(check, ShouldResemble, &dto.TriggerCheck{
			TriggerID: triggerID,
			CheckData: &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"metric2": {},
				},
			},
		})
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Error get")
		dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, expected)
		check, err := GetTriggerLastCheck(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(check, ShouldBeNil)
	})
}

func TestDeleteTriggerThrottling(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func() {
		dataBase.EXPECT().DeleteTriggerThrottling(triggerID).Return(nil)
		var total int64
		var to int64 = -1
		dataBase.EXPECT().GetNotifications(total, to).Return(make([]*moira.ScheduledNotification, 0), total, nil)
		dataBase.EXPECT().AddNotifications(make([]*moira.ScheduledNotification, 0), gomock.Any()).Return(nil)
		err := DeleteTriggerThrottling(dataBase, triggerID)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Error delete")
		dataBase.EXPECT().DeleteTriggerThrottling(triggerID).Return(expected)
		err := DeleteTriggerThrottling(dataBase, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestSetTriggerMaintenance(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.Must(uuid.NewV4()).String()
	metricsMaintenance := dto.MetricsMaintenance{
		"Metric1": 12345,
		"Metric2": 12346,
	}
	triggerMaintenance := dto.TriggerMaintenance{Metrics: map[string]int64(metricsMaintenance)}
	var maintenanceTS int64 = 12347

	Convey("Success setting metrics maintenance only", t, func() {
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(nil)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldBeNil)
	})

	Convey("Success setting trigger maintenance only", t, func() {
		triggerMaintenance.Trigger = &maintenanceTS
		triggerMaintenance.Metrics = dto.MetricsMaintenance{}
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(nil)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldBeNil)
	})

	Convey("Success setting metrics and trigger maintenance at once", t, func() {
		triggerMaintenance.Trigger = &maintenanceTS
		triggerMaintenance.Metrics = metricsMaintenance
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(nil)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Error set")
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(expected)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}
