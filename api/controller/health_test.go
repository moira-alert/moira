package controller

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestGetNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("On startup should return OK", t, func() {
		expectedState := dto.NotifierState{State: moira.SelfStateOK}
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{State: moira.SelfStateOK}, nil)
		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})
}

func TestUpdateNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Setting OK notifier state", t, func() {
		expectedState := dto.NotifierState{State: moira.SelfStateOK}
		now := time.Now().UTC()
		dataBase.EXPECT().SetNotifierState(moira.NotifierState{
			Actor:     moira.SelfStateActorManual,
			State:     moira.SelfStateOK,
			Timestamp: now,
		}).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{State: moira.SelfStateOK}, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateOK}, now)
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

	Convey("Setting ERROR notifier state", t, func() {
		expectedState := dto.NotifierState{State: moira.SelfStateERROR, Message: dto.ErrorMessage}
		now := time.Now().UTC()
		dataBase.EXPECT().SetNotifierState(moira.NotifierState{
			Actor:     moira.SelfStateActorManual,
			State:     moira.SelfStateERROR,
			Timestamp: now,
		}).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{State: moira.SelfStateERROR}, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateERROR}, now)
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})
}
