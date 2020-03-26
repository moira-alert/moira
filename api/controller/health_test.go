package controller

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("On startup should return OK", t, func() {
		expectedState := dto.NotifierState{State: moira.SelfStateOK}
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateOK, moira.SelfStateOKMessage, nil)
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
		expectedState := dto.NotifierState{State: moira.SelfStateOK, Message: moira.SelfStateOKMessage}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateOK, moira.SelfStateOKMessage).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateOK, moira.SelfStateOKMessage, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateOK})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

	Convey("Setting ERROR notifier state with no message", t, func() {
		expectedState := dto.NotifierState{State: moira.SelfStateERROR, Message: moira.SelfStateErrorMessage}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateERROR, moira.SelfStateErrorMessage).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, moira.SelfStateErrorMessage, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateERROR, Message: ""})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})


	Convey("Setting ERROR notifier state with custom error message", t, func() {
		message := "Moira has been turned off for routine maintenance"
		expectedState := dto.NotifierState{State: moira.SelfStateERROR, Message: message}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateERROR, message).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, message, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateERROR, Message: message})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)
		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

}
