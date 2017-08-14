package checker

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
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
		TriggerId: "superId",
		Database:  dataBase,
		Logger:    logger,
	}

	Convey("Test errors", t, func() {
		Convey("Get trigger error", func() {
			getTriggerError := fmt.Errorf("Oppps! Can't read trigger")
			dataBase.EXPECT().GetTrigger(triggerChecker.TriggerId).Return(nil, getTriggerError)
			err := triggerChecker.InitTriggerChecker()
			So(err, ShouldBeError)
			So(err, ShouldResemble, getTriggerError)
		})

		Convey("No trigger error", func() {
			dataBase.EXPECT().GetTrigger(triggerChecker.TriggerId).Return(nil, nil)
			err := triggerChecker.InitTriggerChecker()
			So(err, ShouldBeError)
			So(err, ShouldResemble, ErrTriggerNotExists)
		})

		Convey("Get lastCheck error", func() {
			readLastCheckError := fmt.Errorf("Oppps! Can't read last check")
			dataBase.EXPECT().GetTrigger(triggerChecker.TriggerId).Return(&moira.Trigger{}, nil)
			dataBase.EXPECT().GetTriggerLastCheck(triggerChecker.TriggerId).Return(nil, readLastCheckError)
			err := triggerChecker.InitTriggerChecker()
			So(err, ShouldBeError)
			So(err, ShouldResemble, readLastCheckError)
		})
	})
}
