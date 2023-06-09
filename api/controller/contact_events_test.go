package controller

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/moira-alert/moira"

	"github.com/moira-alert/moira/api"

	"github.com/moira-alert/moira/api/dto"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetContactEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	contact := dto.Contact{
		ID:     "some_contact_id",
		Type:   "telegram",
		Value:  "#some_tg_channel",
		TeamID: "MoiraCoolestTeam",
	}

	contactExpect := moira.ContactData{
		ID:    contact.ID,
		Value: contact.Value,
		User:  "",
		Team:  contact.TeamID,
	}

	items := []*moira.NotificationEventHistoryItem{
		{
			TimeStamp: 924526680,
			Metric:    "some.metric1",
			State:     moira.StateOK,
			OldState:  moira.StateERROR,
			TriggerID: "someTriggerId",
			ContactID: "some_contact_id",
		},

		{
			TimeStamp: 938782680,
			Metric:    "some.metric2s",
			State:     moira.StateWARN,
			OldState:  moira.StateOK,
			TriggerID: "someTriggerId",
			ContactID: "some_contact_id",
		},
	}

	dataBaseSearchFrom := strconv.FormatInt(items[0].TimeStamp, 10)
	dataBaseSearchTo := strconv.FormatInt(items[1].TimeStamp, 10)

	Convey("Ensure that request of contact with events with invalid parameters returns api error", t, func() {

		Convey("in case of both params are empty", func() {
			contactWithEvents, err := GetContactByIdWithEventsLimit(dataBase, contact.ID, "", "")
			So(contactWithEvents, ShouldBeNil)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("'from' and 'to' query params should specified")))
		})

		Convey("in case of 'from' param is empty", func() {
			contactWithEvents, err := GetContactByIdWithEventsLimit(dataBase, contact.ID, "", dataBaseSearchTo)
			So(contactWithEvents, ShouldBeNil)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("'from' and 'to' query params should specified")))
		})

		Convey("in case of 'to' param is empty", func() {
			contactWithEvents, err := GetContactByIdWithEventsLimit(dataBase, contact.ID, dataBaseSearchFrom, "")
			So(contactWithEvents, ShouldBeNil)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("'from' and 'to' query params should specified")))
		})

		Convey("in case of 'to' parameter is not number at all", func() {
			contactWithEvents, err := GetContactByIdWithEventsLimit(dataBase, contact.ID, dataBaseSearchFrom, "not_number_here")
			So(contactWithEvents, ShouldBeNil)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("'from' and 'to' query params should be positive numbers")))
		})
	})

	Convey("Ensure that if we have error while getting contact we will have api error", t, func() {
		dbErr := fmt.Errorf("can't get contact error here")
		apiErr := api.ErrorInternalServer(fmt.Errorf("GetContactByIdWithEventsLimit: can't get contact with id %v", contact.ID))

		dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, dbErr)
		contactWithEvents, apiErrActual := GetContactByIdWithEventsLimit(dataBase, contact.ID, dataBaseSearchFrom, dataBaseSearchTo)

		So(apiErrActual, ShouldResemble, apiErr)
		So(contactWithEvents, ShouldBeNil)
	})

	Convey("Ensure that if we have error while getting events we will have api error", t, func() {
		var emptyNotifications []*moira.NotificationEventHistoryItem
		dbErr := fmt.Errorf("can't get events error here")
		apiErr := api.ErrorInternalServer(fmt.Errorf("GetContactByIdWithEventsLimit: can't get notifications for contact with id %v", contact.ID))

		dataBase.EXPECT().GetContact(contact.ID).Return(contactExpect, nil)
		dataBase.EXPECT().GetNotificationsByContactIdWithLimit(contact.ID, items[0].TimeStamp, items[1].TimeStamp).Return(emptyNotifications, dbErr)

		contactWithEvents, apiErrActual := GetContactByIdWithEventsLimit(dataBase, contact.ID, dataBaseSearchFrom, dataBaseSearchTo)

		So(apiErrActual, ShouldResemble, apiErr)
		So(contactWithEvents, ShouldBeNil)
	})

	Convey("Ensure that everything is fine if db works correctly", t, func() {
		dataBase.EXPECT().GetContact(contact.ID).Return(contactExpect, nil)
		dataBase.EXPECT().GetNotificationsByContactIdWithLimit(contact.ID, items[0].TimeStamp, items[1].TimeStamp).Return(items, nil)

		contactWithEvents, apiErrActual := GetContactByIdWithEventsLimit(dataBase, contact.ID, dataBaseSearchFrom, dataBaseSearchTo)

		So(apiErrActual, ShouldBeNil)
		So(contactWithEvents, ShouldResemble, items)
	})
}
