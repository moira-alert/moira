package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestUpdateTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success update", t, func() {
		triggerModel := dto.TriggerModel{ID: uuid.Must(uuid.NewV4()).String()}
		trigger := triggerModel.ToMoiraTrigger()
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(*trigger, nil)
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 30)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), trigger.ClusterKey()).Return(nil)
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

	Convey("With no existing trigger", t, func() {
		Convey("No timeSeries", func() {
			Convey("No last check", func() {
				dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
				dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
				dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
				dataBase.EXPECT().
					SetTriggerLastCheck(
						triggerID,
						&moira.CheckData{
							Metrics: make(map[string]moira.MetricState),
							State:   moira.StateNODATA,
							Score:   1000,
						},
						trigger.ClusterKey()).
					Return(nil)
				dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
				resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
				So(err, ShouldBeNil)
				So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
			})
			Convey("Has last check", func() {
				actualLastCheck := lastCheck

				dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
				dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
				dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(actualLastCheck, nil)
				dataBase.EXPECT().SetTriggerLastCheck(triggerID, &emptyLastCheck, trigger.ClusterKey()).Return(nil)
				dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
				resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
				So(err, ShouldBeNil)
				So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
			})
		})

		Convey("Has timeSeries", func() {
			Convey("No last check", func() {
				dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
				dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
				dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
				dataBase.EXPECT().
					SetTriggerLastCheck(
						triggerID,
						&moira.CheckData{
							Metrics: make(map[string]moira.MetricState),
							State:   moira.StateNODATA,
							Score:   1000,
						},
						trigger.ClusterKey()).
					Return(nil)
				dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
				resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, map[string]bool{"super.metric1": true, "super.metric2": true})
				So(err, ShouldBeNil)
				So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
			})

			Convey("Has last check", func() {
				actualLastCheck := moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"super.metric1": {},
						"super.metric2": {},
					},
					MetricsToTargetRelation: map[string]string{
						"t2": "super.metric3",
					},
				}
				expectedLastCheck := &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"super.metric1": {},
						"super.metric2": {},
					},
					MetricsToTargetRelation: make(map[string]string),
				}

				dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
				dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
				dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(actualLastCheck, nil)
				dataBase.EXPECT().SetTriggerLastCheck(triggerID, expectedLastCheck, trigger.ClusterKey()).Return(nil)
				dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
				resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, map[string]bool{"super.metric1": true, "super.metric2": true})
				So(err, ShouldBeNil)
				So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
			})
		})
	})

	Convey("Errors", t, func() {
		Convey("AcquireTriggerCheckLock error", func() {
			expected := fmt.Errorf("acquireTriggerCheckLock error")
			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30).Return(expected)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("GetTriggerLastCheck error", func() {
			expected := fmt.Errorf("getTriggerLastCheck error")

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, expected)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("SetTriggerLastCheck error", func() {
			expected := fmt.Errorf("setTriggerLastCheck error")

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any(), trigger.ClusterKey()).Return(expected)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("saveTrigger error", func() {
			expected := fmt.Errorf("saveTrigger error")

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any(), trigger.ClusterKey()).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(expected)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
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

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.ClusterKey()).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("ERROR TTLState", func() {
			trigger.TTLState = &moira.TTLStateERROR
			lastCheck.State = moira.StateERROR
			lastCheck.Score = 100

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.ClusterKey()).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("WARN TTLState", func() {
			trigger.TTLState = &moira.TTLStateWARN
			lastCheck.State = moira.StateWARN
			lastCheck.Score = 1

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.ClusterKey()).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("OK TTLState", func() {
			trigger.TTLState = &moira.TTLStateOK
			lastCheck.State = moira.StateOK
			lastCheck.Score = 0

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.ClusterKey()).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})

		Convey("DEL TTLState", func() {
			trigger.TTLState = &moira.TTLStateDEL
			lastCheck.State = moira.StateOK
			lastCheck.Score = 0

			dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
			dataBase.EXPECT().DeleteTriggerCheckLock(triggerID)
			dataBase.EXPECT().GetTriggerLastCheck(triggerID).Return(moira.CheckData{}, database.ErrNil)
			dataBase.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck, trigger.ClusterKey()).Return(nil)
			dataBase.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := saveTrigger(dataBase, nil, &trigger, triggerID, make(map[string]bool))
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

	Convey("Returns all metrics, because their DeletedButKept is false", t, func() {
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

	Convey("Does not return all metrics, as some DeletedButKept is true", t, func() {
		lastCheck = moira.CheckData{
			Metrics: map[string]moira.MetricState{
				"metric": {
					DeletedButKept: true,
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
		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(nil)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldBeNil)
	})

	Convey("Success setting trigger maintenance only", t, func() {
		triggerMaintenance.Trigger = &maintenanceTS
		triggerMaintenance.Metrics = dto.MetricsMaintenance{}

		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(nil)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldBeNil)
	})

	Convey("Success setting metrics and trigger maintenance at once", t, func() {
		triggerMaintenance.Trigger = &maintenanceTS
		triggerMaintenance.Metrics = metricsMaintenance

		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(nil)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Error set")

		dataBase.EXPECT().AcquireTriggerCheckLock(triggerID, 30)
		dataBase.EXPECT().ReleaseTriggerCheckLock(triggerID)
		dataBase.EXPECT().SetTriggerCheckMaintenance(triggerID, triggerMaintenance.Metrics, triggerMaintenance.Trigger, "", int64(0)).Return(expected)
		err := SetTriggerMaintenance(dataBase, triggerID, triggerMaintenance, "", 0)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func Test_metricEvaluationRulesChanged(t *testing.T) {
	Convey("Test metricEvaluationRulesChanged", t, func() {
		type testcase struct {
			desc            string
			givenOldTrigger *moira.Trigger
			givenNewTrigger *moira.Trigger
			expectedResult  bool
		}

		var (
			floatValueOne                = 1.0
			floatValueEqualToValueOne    = 1.0
			floatValueNotEqualToValueOne = 2.0

			stringValueOne                = "some str"
			stringValueEqualToValueOne    = "some str"
			stringValueNotEqualToValueOne = "another str"
		)

		cases := []testcase{
			{
				desc:            "with nil existed trigger",
				givenOldTrigger: nil,
				givenNewTrigger: &moira.Trigger{},
				expectedResult:  true,
			},
			{
				desc:            "with different number of targets",
				givenOldTrigger: &moira.Trigger{Targets: []string{"hello"}},
				givenNewTrigger: &moira.Trigger{Targets: []string{"user", "bye"}},
				expectedResult:  true,
			},
			{
				desc:            "with different targets",
				givenOldTrigger: &moira.Trigger{Targets: []string{"hello", "mama"}},
				givenNewTrigger: &moira.Trigger{Targets: []string{"user", "bye"}},
				expectedResult:  true,
			},
			{
				desc:            "with different trigger type",
				givenOldTrigger: &moira.Trigger{TriggerType: moira.ExpressionTrigger},
				givenNewTrigger: &moira.Trigger{TriggerType: moira.FallingTrigger},
				expectedResult:  true,
			},
			{
				desc:            "with warn value not set for one",
				givenOldTrigger: &moira.Trigger{WarnValue: nil},
				givenNewTrigger: &moira.Trigger{WarnValue: &floatValueOne},
				expectedResult:  true,
			},
			{
				desc:            "with warn value not set for other",
				givenOldTrigger: &moira.Trigger{WarnValue: &floatValueOne},
				givenNewTrigger: &moira.Trigger{WarnValue: nil},
				expectedResult:  true,
			},
			{
				desc:            "with different warn values",
				givenOldTrigger: &moira.Trigger{WarnValue: &floatValueOne},
				givenNewTrigger: &moira.Trigger{WarnValue: &floatValueNotEqualToValueOne},
				expectedResult:  true,
			},
			{
				desc:            "with same warn values",
				givenOldTrigger: &moira.Trigger{WarnValue: &floatValueOne},
				givenNewTrigger: &moira.Trigger{WarnValue: &floatValueEqualToValueOne},
				expectedResult:  false,
			},
			{
				desc:            "with error value not set for one",
				givenOldTrigger: &moira.Trigger{ErrorValue: nil},
				givenNewTrigger: &moira.Trigger{ErrorValue: &floatValueOne},
				expectedResult:  true,
			},
			{
				desc:            "with error value not set for other",
				givenOldTrigger: &moira.Trigger{ErrorValue: &floatValueOne},
				givenNewTrigger: &moira.Trigger{ErrorValue: nil},
				expectedResult:  true,
			},
			{
				desc:            "with different error values",
				givenOldTrigger: &moira.Trigger{ErrorValue: &floatValueOne},
				givenNewTrigger: &moira.Trigger{ErrorValue: &floatValueNotEqualToValueOne},
				expectedResult:  true,
			},
			{
				desc:            "with same error values",
				givenOldTrigger: &moira.Trigger{ErrorValue: &floatValueOne},
				givenNewTrigger: &moira.Trigger{ErrorValue: &floatValueEqualToValueOne},
				expectedResult:  false,
			},
			{
				desc:            "with ttl state not set for one",
				givenOldTrigger: &moira.Trigger{TTLState: nil},
				givenNewTrigger: &moira.Trigger{TTLState: &moira.TTLStateNODATA},
				expectedResult:  true,
			},
			{
				desc:            "with ttl state not set for other",
				givenOldTrigger: &moira.Trigger{TTLState: &moira.TTLStateNODATA},
				givenNewTrigger: &moira.Trigger{TTLState: nil},
				expectedResult:  true,
			},
			{
				desc:            "with different ttl states",
				givenOldTrigger: &moira.Trigger{TTLState: &moira.TTLStateNODATA},
				givenNewTrigger: &moira.Trigger{TTLState: &moira.TTLStateERROR},
				expectedResult:  true,
			},
			{
				desc:            "with same ttl state",
				givenOldTrigger: &moira.Trigger{TTLState: &moira.TTLStateNODATA},
				givenNewTrigger: &moira.Trigger{TTLState: &moira.TTLStateNODATA},
				expectedResult:  false,
			},
			{
				desc:            "with expression not set for one",
				givenOldTrigger: &moira.Trigger{Expression: nil},
				givenNewTrigger: &moira.Trigger{Expression: &stringValueOne},
				expectedResult:  true,
			},
			{
				desc:            "with expression not set for other",
				givenOldTrigger: &moira.Trigger{Expression: &stringValueOne},
				givenNewTrigger: &moira.Trigger{Expression: nil},
				expectedResult:  true,
			},
			{
				desc:            "with different expressions",
				givenOldTrigger: &moira.Trigger{Expression: &stringValueOne},
				givenNewTrigger: &moira.Trigger{Expression: &stringValueNotEqualToValueOne},
				expectedResult:  true,
			},
			{
				desc:            "with same expression",
				givenOldTrigger: &moira.Trigger{Expression: &stringValueOne},
				givenNewTrigger: &moira.Trigger{Expression: &stringValueEqualToValueOne},
				expectedResult:  false,
			},
			{
				desc:            "with different trigger source",
				givenOldTrigger: &moira.Trigger{TriggerSource: moira.PrometheusRemote},
				givenNewTrigger: &moira.Trigger{TriggerSource: moira.GraphiteLocal},
				expectedResult:  true,
			},
			{
				desc:            "with different cluster id",
				givenOldTrigger: &moira.Trigger{ClusterId: moira.DefaultCluster},
				givenNewTrigger: &moira.Trigger{ClusterId: moira.ClusterNotSet},
				expectedResult:  true,
			},
			{
				desc:            "with different number of alone metrics",
				givenOldTrigger: &moira.Trigger{AloneMetrics: map[string]bool{"t1": true, "t2": true}},
				givenNewTrigger: &moira.Trigger{AloneMetrics: map[string]bool{"t1": true, "t2": true, "t3": true}},
				expectedResult:  true,
			},
			{
				desc:            "with different alone metrics",
				givenOldTrigger: &moira.Trigger{AloneMetrics: map[string]bool{"t1": true, "t2": true}},
				givenNewTrigger: &moira.Trigger{AloneMetrics: map[string]bool{"t1": true, "t3": true}},
				expectedResult:  true,
			},
		}

		for i, tc := range cases {
			Convey(fmt.Sprintf("Case %v: %s", i+1, tc.desc), func() {
				So(metricEvaluationRulesChanged(tc.givenOldTrigger, tc.givenNewTrigger),
					ShouldResemble,
					tc.expectedResult)
			})
		}
	})
}
