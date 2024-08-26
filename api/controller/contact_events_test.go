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

func TestGetContactEventsByIdWithLimit(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	now := time.Now()

	contact := dto.Contact{
		ID:     "some_contact_id",
		Type:   "telegram",
		Value:  "#some_tg_channel",
		TeamID: "MoiraCoolestTeam",
	}

	contactExpect := moira.ContactData{
		ID:    contact.ID,
		Value: contact.Value,
		Type:  contact.Type,
		User:  "",
		Team:  contact.TeamID,
	}

	items := []*moira.NotificationEventHistoryItem{
		{
			TimeStamp: now.Unix() - 20,
			Metric:    "some.metric1",
			State:     moira.StateOK,
			OldState:  moira.StateERROR,
			TriggerID: "someTriggerId",
			ContactID: "some_contact_id",
		},

		{
			TimeStamp: now.Unix() - 50,
			Metric:    "some.metric2",
			State:     moira.StateWARN,
			OldState:  moira.StateOK,
			TriggerID: "someTriggerId",
			ContactID: "some_contact_id",
		},
	}

	itemsExpected := dto.ContactEventItemList{
		List: []dto.ContactEventItem{
			{
				TimeStamp: now.Unix() - 20,
				Metric:    "some.metric1",
				State:     "OK",
				OldState:  "ERROR",
				TriggerID: "someTriggerId",
			},
			{
				TimeStamp: now.Unix() - 50,
				Metric:    "some.metric2",
				State:     "WARN",
				OldState:  "OK",
				TriggerID: "someTriggerId",
			},
		},
	}

	defaultToParameter := now.Unix()
	defaultFromParameter := defaultToParameter - int64((3 * time.Hour).Seconds())
	defaultPage := int64(0)
	defaultSize := int64(100)

	Convey("Ensure that request with default parameters would return both event items (no url params specified)", t, func() {
		dataBase.EXPECT().GetContact(contact.ID).Return(contactExpect, nil).AnyTimes()
		dataBase.EXPECT().GetNotificationsHistoryByContactID(contact.ID, defaultFromParameter, defaultToParameter, defaultPage, defaultSize).Return(items, nil)

		actualEvents, err := GetContactEventsHistoryByID(dataBase, contact.ID, defaultFromParameter, defaultToParameter, defaultPage, defaultSize)

		So(err, ShouldBeNil)
		So(actualEvents, ShouldResemble, &itemsExpected)
	})

	Convey("Ensure that request with only 'from' parameter given and 'to' default will return only one (newest) event", t, func() {
		dataBase.EXPECT().GetContact(contact.ID).Return(contactExpect, nil).AnyTimes()
		dataBase.EXPECT().GetNotificationsHistoryByContactID(contact.ID, defaultFromParameter-20, defaultToParameter, defaultPage, defaultSize).Return(items[:1], nil)

		actualEvents, err := GetContactEventsHistoryByID(dataBase, contact.ID, defaultFromParameter-20, defaultToParameter, defaultPage, defaultSize)
		So(err, ShouldBeNil)
		So(actualEvents, ShouldResemble, &dto.ContactEventItemList{
			List: []dto.ContactEventItem{
				itemsExpected.List[0],
			},
		})
	})

	Convey("Ensure that request with only 'to' parameter given and 'from' default will return only one (oldest) event", t, func() {
		dataBase.EXPECT().GetContact(contact.ID).Return(contactExpect, nil).AnyTimes()
		dataBase.EXPECT().GetNotificationsHistoryByContactID(contact.ID, defaultFromParameter, defaultToParameter-30, defaultPage, defaultSize).Return(items[1:], nil)

		actualEvents, err := GetContactEventsHistoryByID(dataBase, contact.ID, defaultFromParameter, defaultToParameter-30, defaultPage, defaultSize)
		So(err, ShouldBeNil)
		So(actualEvents, ShouldResemble, &dto.ContactEventItemList{
			List: []dto.ContactEventItem{
				itemsExpected.List[1],
			},
		})
	})
}
