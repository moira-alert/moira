package controller

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	triggerID := uuid.NewV4().String()
	var page int64 = 10
	var size int64 = 100

	Convey("Test has events", t, func() {
		var total int64 = 6000000
		dataBase.EXPECT().GetEvents(triggerID, page*size, size-1).Return([]*moira.EventData{{State: "NODATA", OldState: "OK"}, {State: "OK", OldState: "NODATA"}}, nil)
		dataBase.EXPECT().GetTriggerEventsCount(triggerID, int64(-1)).Return(total)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.EventsList{
			List:  []*moira.EventData{{State: "NODATA", OldState: "OK"}, {State: "OK", OldState: "NODATA"}},
			Total: total,
			Size:  size,
			Page:  page,
		})
	})

	Convey("Test no events", t, func() {
		var total int64
		dataBase.EXPECT().GetEvents(triggerID, page*size, size-1).Return(make([]*moira.EventData, 0), nil)
		dataBase.EXPECT().GetTriggerEventsCount(triggerID, int64(-1)).Return(total)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.EventsList{
			List:  make([]*moira.EventData, 0),
			Total: total,
			Size:  size,
			Page:  page,
		})
	})

	Convey("Test error", t, func() {
		expected := fmt.Errorf("Oooops! Can not get all contacts")
		dataBase.EXPECT().GetEvents(triggerID, page*size, size-1).Return(nil, expected)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}
