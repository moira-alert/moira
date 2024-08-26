package controller

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

var (
	allMetrics = regexp.MustCompile(``)
	allStates  map[string]struct{}
)

func TestGetEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	triggerID := uuid.Must(uuid.NewV4()).String()

	var page int64 = 10
	var size int64 = 100
	from := "-inf"
	to := "+inf"

	Convey("Test has events", t, func() {
		var total int64 = 6000000
		dataBase.EXPECT().GetNotificationEvents(triggerID, page, size, from, to).
			Return([]*moira.NotificationEvent{
				{
					State:    moira.StateNODATA,
					OldState: moira.StateOK,
				},
				{
					State:    moira.StateOK,
					OldState: moira.StateNODATA,
				},
			}, nil)
		dataBase.EXPECT().GetNotificationEventCount(triggerID, int64(-1)).Return(total)

		list, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, allMetrics, allStates)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.EventsList{
			List:  []moira.NotificationEvent{{State: moira.StateNODATA, OldState: moira.StateOK}, {State: moira.StateOK, OldState: moira.StateNODATA}},
			Total: total,
			Size:  size,
			Page:  page,
		})
	})

	Convey("Test no events", t, func() {
		var total int64
		dataBase.EXPECT().GetNotificationEvents(triggerID, page, size, from, to).Return(make([]*moira.NotificationEvent, 0), nil)
		dataBase.EXPECT().GetNotificationEventCount(triggerID, int64(-1)).Return(total)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, allMetrics, allStates)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.EventsList{
			List:  make([]moira.NotificationEvent, 0),
			Total: total,
			Size:  size,
			Page:  page,
		})
	})

	Convey("Test error", t, func() {
		expected := fmt.Errorf("oooops! Can not get all contacts")
		dataBase.EXPECT().GetNotificationEvents(triggerID, page, size, from, to).Return(nil, expected)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, allMetrics, allStates)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("Test filtering", t, func() {
		Convey("by metric regex", func() {
			page = 0
			size = 2
			Convey("with same pattern", func() {
				filtered := []*moira.NotificationEvent{
					{Metric: "metric.test.event1"},
					{Metric: "a.metric.test.event2"},
				}
				notFiltered := []*moira.NotificationEvent{
					{Metric: "another.mEtric.test.event"},
					{Metric: "metric.test"},
				}
				firstPortion := append(make([]*moira.NotificationEvent, 0), notFiltered[0], filtered[0])
				dataBase.EXPECT().GetNotificationEvents(triggerID, page, size, from, to).Return(firstPortion, nil)

				secondPortion := append(make([]*moira.NotificationEvent, 0), filtered[1], notFiltered[1])
				dataBase.EXPECT().GetNotificationEvents(triggerID, page+1, size, from, to).Return(secondPortion, nil)

				total := int64(len(firstPortion) + len(secondPortion))
				dataBase.EXPECT().GetNotificationEventCount(triggerID, int64(-1)).Return(total)

				actual, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, regexp.MustCompile(`metric\.test\.event`), allStates)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, &dto.EventsList{
					Page:  page,
					Size:  size,
					Total: total,
					List:  toDTOList(filtered),
				})
			})
		})
		page = 0
		size = -1

		Convey("by state", func() {
			filtered := []*moira.NotificationEvent{
				{State: moira.StateOK},
				{State: moira.StateTEST},
				{State: moira.StateEXCEPTION},
			}
			notFiltered := []*moira.NotificationEvent{
				{State: moira.StateWARN},
				{State: moira.StateNODATA},
				{State: moira.StateERROR},
			}
			Convey("with empty map all allowed", func() {
				total := int64(len(filtered) + len(notFiltered))
				dataBase.EXPECT().GetNotificationEvents(triggerID, page, size, from, to).Return(append(filtered, notFiltered...), nil)
				dataBase.EXPECT().GetNotificationEventCount(triggerID, int64(-1)).Return(total)

				actual, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, allMetrics, allStates)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, &dto.EventsList{
					Page:  page,
					Size:  size,
					Total: total,
					List:  toDTOList(append(filtered, notFiltered...)),
				})
			})

			Convey("with given states", func() {
				total := int64(len(filtered) + len(notFiltered))
				dataBase.EXPECT().GetNotificationEvents(triggerID, page, size, from, to).Return(append(filtered, notFiltered...), nil)
				dataBase.EXPECT().GetNotificationEventCount(triggerID, int64(-1)).Return(total)

				actual, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, allMetrics, map[string]struct{}{
					string(moira.StateOK):        {},
					string(moira.StateEXCEPTION): {},
					string(moira.StateTEST):      {},
				})
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, &dto.EventsList{
					Page:  page,
					Size:  size,
					Total: total,
					List:  toDTOList(filtered),
				})
			})
		})
	})
}

func toDTOList(eventPtrs []*moira.NotificationEvent) []moira.NotificationEvent {
	events := make([]moira.NotificationEvent, 0, len(eventPtrs))
	for _, ptr := range eventPtrs {
		events = append(events, *ptr)
	}
	return events
}

func TestDeleteAllNotificationEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Success", t, func() {
		dataBase.EXPECT().RemoveAllNotificationEvents().Return(nil)
		err := DeleteAllEvents(dataBase)
		So(err, ShouldBeNil)
	})

	Convey("Error delete", t, func() {
		expected := fmt.Errorf("oooops! Can not get notifications")
		dataBase.EXPECT().RemoveAllNotificationEvents().Return(expected)
		err := DeleteAllEvents(dataBase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}
