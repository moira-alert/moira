package controller

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/notifier/selfstate"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("On startup should return OK", t, func() {
		expectedState := dto.NotifierState{State: selfstate.OK}
		dataBase.EXPECT().GetNotifierState().Return(selfstate.OK, nil)
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
		expectedState := dto.NotifierState{State: selfstate.OK}
		dataBase.EXPECT().SetNotifierState(selfstate.OK).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(selfstate.OK, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: selfstate.OK})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

	Convey("Setting ERROR notifier state", t, func() {
		expectedState := dto.NotifierState{State: selfstate.ERROR, Message: dto.ErrorMessage}
		dataBase.EXPECT().SetNotifierState(selfstate.ERROR).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(selfstate.ERROR, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: selfstate.ERROR})
		So(err, ShouldBeNil)

		actualState, err := GetNotifierState(dataBase)

		So(*actualState, ShouldResemble, expectedState)
		So(err, ShouldBeNil)
	})

}
