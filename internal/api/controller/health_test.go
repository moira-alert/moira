package controller

import (
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/internal/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/internal/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("On startup should return OK", t, func() {
		expectedState := dto.NotifierState{State: moira2.SelfStateOK}
		dataBase.EXPECT().GetNotifierState().Return(moira2.SelfStateOK, nil)
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
		expectedState := dto.NotifierState{State: moira2.SelfStateOK}
		dataBase.EXPECT().SetNotifierState(moira2.SelfStateOK).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira2.SelfStateOK, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira2.SelfStateOK})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

	Convey("Setting ERROR notifier state", t, func() {
		expectedState := dto.NotifierState{State: moira2.SelfStateERROR, Message: dto.ErrorMessage}
		dataBase.EXPECT().SetNotifierState(moira2.SelfStateERROR).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira2.SelfStateERROR, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira2.SelfStateERROR})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

}
