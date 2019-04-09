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

	Convey("On startup should return OK", t, func(c C) {
		expectedState := dto.NotifierState{State: moira.SelfStateOK}
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)
		actualState, err := GetNotifierState(dataBase)

		c.So(*actualState, ShouldResemble, expectedState)
		c.So(err, ShouldBeNil)
	})
}

func TestUpdateNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Setting OK notifier state", t, func(c C) {
		expectedState := dto.NotifierState{State: moira.SelfStateOK}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateOK).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateOK})
		c.So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		c.So(*actualState, ShouldResemble, expectedState)
		c.So(err, ShouldBeNil)
	})

	Convey("Setting ERROR notifier state", t, func(c C) {
		expectedState := dto.NotifierState{State: moira.SelfStateERROR, Message: dto.ErrorMessage}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateERROR})
		c.So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		c.So(*actualState, ShouldResemble, expectedState)
		c.So(err, ShouldBeNil)
	})

}
