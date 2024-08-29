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

	var page int64 = 1
	var size int64 = 2
	from := "-inf"
	to := "+inf"

	Convey("Test has events", t, func() {
		events := []*moira.NotificationEvent{
			{
				State:    moira.StateNODATA,
				OldState: moira.StateOK,
			},
			{
				State:    moira.StateOK,
				OldState: moira.StateNODATA,
			},
			{
				State:    moira.StateWARN,
				OldState: moira.StateOK,
			},
			{
				State:    moira.StateERROR,
				OldState: moira.StateWARN,
			},
		}
		dataBase.EXPECT().GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to).
			Return(events, nil)

		list, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, allMetrics, allStates)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.EventsList{
			List: []moira.NotificationEvent{
				*events[2],
				*events[3],
			},
			Total: int64(len(events)),
			Size:  size,
			Page:  page,
		})
	})

	Convey("Test no events", t, func() {
		var total int64
		dataBase.EXPECT().GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to).Return(make([]*moira.NotificationEvent, 0), nil)
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
		dataBase.EXPECT().GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to).Return(nil, expected)
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
					{Metric: "metric.test.event.other"},
				}
				notFiltered := []*moira.NotificationEvent{
					{Metric: "another.mEtric.test.event"},
					{Metric: "metric.test"},
				}

				events := []*moira.NotificationEvent{
					notFiltered[0],
					filtered[0],
					notFiltered[1],
					filtered[1],
					filtered[2],
				}
				dataBase.EXPECT().GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to).Return(events, nil)

				total := int64(len(filtered))

				actual, err := GetTriggerEvents(dataBase, triggerID, page, size, from, to, regexp.MustCompile(`metric\.test\.event`), allStates)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, &dto.EventsList{
					Page:  page,
					Size:  size,
					Total: total,
					List:  toDTOList(filtered[:size]),
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
				dataBase.EXPECT().GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to).Return(append(filtered, notFiltered...), nil)

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
				total := int64(len(filtered))
				dataBase.EXPECT().GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to).Return(append(filtered, notFiltered...), nil)

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

	Convey("test paginating", t, func() {
		events := []*moira.NotificationEvent{
			{
				State:    moira.StateNODATA,
				OldState: moira.StateOK,
			},
			{
				State:    moira.StateOK,
				OldState: moira.StateNODATA,
			},
			{
				State:    moira.StateWARN,
				OldState: moira.StateOK,
			},
			{
				State:    moira.StateERROR,
				OldState: moira.StateWARN,
			},
		}
		total := int64(len(events))

		type testcase struct {
			description    string
			expectedEvents []moira.NotificationEvent
			givenPage      int64
			givenSize      int64
		}

		testcases := []testcase{
			{
				description: "with page > 0 and size > 0",
				givenPage:   1,
				givenSize:   1,
				expectedEvents: []moira.NotificationEvent{
					*events[1],
				},
			},
			{
				description: "with page == 0 and size > 0",
				givenPage:   0,
				givenSize:   1,
				expectedEvents: []moira.NotificationEvent{
					*events[0],
				},
			},
			{
				description: "with page > 0, size > 0, page * size + size > events count",
				givenPage:   1,
				givenSize:   3,
				expectedEvents: []moira.NotificationEvent{
					*events[3],
				},
			},
			{
				description:    "with page = 0, size < 0 fetch all events",
				givenPage:      0,
				givenSize:      -10,
				expectedEvents: toDTOList(events),
			},
			{
				description:    "with page > 0, size < 0 return no events",
				givenPage:      1,
				givenSize:      -1,
				expectedEvents: []moira.NotificationEvent{},
			},
			{
				description:    "with page < 0 return no events",
				givenPage:      -1,
				givenSize:      1,
				expectedEvents: []moira.NotificationEvent{},
			},
		}

		for i := range testcases {
			Convey(fmt.Sprintf("test case %d: %s", i+1, testcases[i].description), func() {
				dataBase.EXPECT().GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to).Return(events, nil)

				actual, err := GetTriggerEvents(dataBase, triggerID, testcases[i].givenPage, testcases[i].givenSize, from, to, allMetrics, allStates)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, &dto.EventsList{
					Page:  testcases[i].givenPage,
					Size:  testcases[i].givenSize,
					Total: total,
					List:  testcases[i].expectedEvents,
				})
			})
		}
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
