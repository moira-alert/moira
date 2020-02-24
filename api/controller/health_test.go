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
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)
		dataBase.EXPECT().GetNotifierMessage().Return(dto.DefaultMessage, nil)
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
		expectedState := dto.NotifierState{State: moira.SelfStateOK, Message: dto.DefaultMessage}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateOK).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)
		dataBase.EXPECT().SetNotifierMessage(dto.DefaultMessage).Return(nil)
		dataBase.EXPECT().GetNotifierMessage().Return(dto.DefaultMessage, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateOK})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

	Convey("Setting ERROR notifier state with default message", t, func() {
		expectedState := dto.NotifierState{State: moira.SelfStateERROR, Message: dto.ErrorMessage}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, nil)
		dataBase.EXPECT().SetNotifierMessage(dto.ErrorMessage).Return(nil)
		dataBase.EXPECT().GetNotifierMessage().Return(dto.ErrorMessage, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateERROR})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

	Convey("Setting ERROR notifier state with user-defined message", t, func() {
		expectedState := dto.NotifierState{State: moira.SelfStateERROR, Message: "Y u no work"}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, nil)
		dataBase.EXPECT().SetNotifierMessage("Y u no work").Return(nil)
		dataBase.EXPECT().GetNotifierMessage().Return("Y u no work", nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateERROR, Message: "Y u no work"})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)
		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

}
