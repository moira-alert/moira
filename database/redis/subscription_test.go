package redis

import (
	"testing"

	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
)

func TestSubscriptionData(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("SubscriptionData manipulation", t, func() {
		Convey("Save-get-remove subscription", func() {
			sub := subscriptions[0]
			Convey("No subscription", func() {
				actual, err := dataBase.GetSubscription(sub.ID)
				So(err, ShouldBeError)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldResemble, moira.SubscriptionData{ThrottlingEnabled: true})
			})
			Convey("Save subscription", func() {
				err := dataBase.SaveSubscription(sub)
				So(err, ShouldBeNil)
			})
			Convey("Get subscription by id, user and tags", func() {
				actual, err := dataBase.GetSubscription(sub.ID)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, *sub)

				actual1, err := dataBase.GetSubscriptions([]string{sub.ID})
				So(err, ShouldBeNil)
				So(actual1, ShouldResemble, []*moira.SubscriptionData{sub})

				actual2, err := dataBase.GetTagsSubscriptions([]string{tag1})
				So(err, ShouldBeNil)
				So(actual2, ShouldResemble, []*moira.SubscriptionData{sub})

				actual3, err := dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
				So(err, ShouldBeNil)
				So(actual3, ShouldResemble, []*moira.SubscriptionData{sub})

				actual4, err := dataBase.GetUserSubscriptionIDs(user1)
				So(err, ShouldBeNil)
				So(actual4, ShouldResemble, []string{sub.ID})
			})

			Convey("Remove sub", func() {
				err := dataBase.RemoveSubscription(sub.ID)
				So(err, ShouldBeNil)
			})
			Convey("Get subscription by id, user and tags, should be empty", func() {
				actual, err := dataBase.GetSubscription(sub.ID)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldResemble, moira.SubscriptionData{ThrottlingEnabled: true})

				actual1, err := dataBase.GetSubscriptions([]string{sub.ID})
				So(err, ShouldBeNil)
				So(actual1, ShouldResemble, []*moira.SubscriptionData{nil})

				actual3, err := dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
				So(err, ShouldBeNil)
				So(actual3, ShouldResemble, []*moira.SubscriptionData{})

				actual4, err := dataBase.GetUserSubscriptionIDs(user1)
				So(err, ShouldBeNil)
				So(actual4, ShouldResemble, []string{})
			})
		})

		Convey("Save batches and remove and check", func() {
			ids := make([]string, len(subscriptions))
			for i, sub := range subscriptions {
				ids[i] = sub.ID
			}

			err := dataBase.SaveSubscriptions(subscriptions)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetSubscriptions(ids)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, subscriptions)

			actual1, err := dataBase.GetUserSubscriptionIDs(user1)
			So(err, ShouldBeNil)
			So(actual1, ShouldHaveLength, len(ids))

			err = dataBase.RemoveSubscription(ids[0])
			So(err, ShouldBeNil)

			actual, err = dataBase.GetSubscriptions(ids)
			So(err, ShouldBeNil)
			So(actual, ShouldHaveLength, len(ids))

			actual1, err = dataBase.GetUserSubscriptionIDs(user1)
			So(err, ShouldBeNil)
			So(actual1, ShouldHaveLength, len(ids)-1)
		})

		Convey("Test rewrite subscription", func() {
			dataBase.flush()
			sub := *subscriptions[0]

			err := dataBase.SaveSubscription(&sub)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetSubscription(sub.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, sub)

			actual1, err := dataBase.GetUserSubscriptionIDs(user1)
			So(err, ShouldBeNil)
			So(actual1, ShouldHaveLength, 1)

			sub.User = user2

			err = dataBase.SaveSubscription(&sub)
			So(err, ShouldBeNil)

			actual, err = dataBase.GetSubscription(sub.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, sub)

			actual1, err = dataBase.GetUserSubscriptionIDs(user1)
			So(err, ShouldBeNil)
			So(actual1, ShouldHaveLength, 0)

			actual1, err = dataBase.GetUserSubscriptionIDs(user2)
			So(err, ShouldBeNil)
			So(actual1, ShouldHaveLength, 1)

			actual3, err := dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
			So(err, ShouldBeNil)
			So(actual3, ShouldResemble, []*moira.SubscriptionData{&sub})

			actual4, err := dataBase.GetTagsSubscriptions([]string{tag1, tag3})
			So(err, ShouldBeNil)
			So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag2})
			So(err, ShouldBeNil)
			So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})

			sub.Tags = []string{tag1, tag3}

			err = dataBase.SaveSubscription(&sub)
			So(err, ShouldBeNil)

			actual, err = dataBase.GetSubscription(sub.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, sub)

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
			So(err, ShouldBeNil)
			So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag2})
			So(err, ShouldBeNil)
			So(actual4, ShouldResemble, []*moira.SubscriptionData{})

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag1, tag3})
			So(err, ShouldBeNil)
			So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})
		})
	})
}

func TestSubscriptionErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		actual1, err := dataBase.GetSubscription("123")
		So(actual1, ShouldResemble, moira.SubscriptionData{ThrottlingEnabled: true})
		So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetSubscriptions([]string{"123"})
		So(actual2, ShouldBeNil)
		So(err, ShouldNotBeNil)

		err = dataBase.SaveSubscriptions(subscriptions)
		So(err, ShouldNotBeNil)

		err = dataBase.SaveSubscription(subscriptions[0])
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveSubscription(subscriptions[0].ID)
		So(err, ShouldNotBeNil)

		actual3, err := dataBase.GetUserSubscriptionIDs("a21213")
		So(actual3, ShouldBeNil)
		So(err, ShouldNotBeNil)

		actual4, err := dataBase.GetTagsSubscriptions([]string{"123"})
		So(actual4, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}

var tag1 = "tag1"
var tag2 = "tag2"
var tag3 = "tag3"

var subscriptions = []*moira.SubscriptionData{
	{
		ID:                "subscriptionID-00000000000001",
		Enabled:           true,
		Tags:              []string{tag1, tag2, tag3},
		Contacts:          []string{uuid.NewV4().String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
	{
		ID:       "subscriptionID-00000000000002",
		Enabled:  true,
		Tags:     []string{tag1},
		Contacts: []string{uuid.NewV4().String()},
		User:     user1,
		Schedule: moira.ScheduleData{
			StartOffset:    10,
			EndOffset:      20,
			TimezoneOffset: 0,
			Days: []moira.ScheduleDataDay{
				{Enabled: false},
				{Enabled: true}, // Tuesday 00:10 - 00:20
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
			},
		},
		ThrottlingEnabled: true,
	},
	{
		ID:       "subscriptionID-00000000000003",
		Enabled:  true,
		Tags:     []string{tag3, tag1},
		Contacts: []string{uuid.NewV4().String()},
		User:     user1,
		Schedule: moira.ScheduleData{
			StartOffset:    0,   // 0:00 (GMT +5) after
			EndOffset:      900, // 15:00 (GMT +5)
			TimezoneOffset: -300,
			Days: []moira.ScheduleDataDay{
				{Enabled: false},
				{Enabled: false},
				{Enabled: true},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
			},
		},
		ThrottlingEnabled: true,
	},
	{
		ID:       "subscriptionID-00000000000004",
		Enabled:  true,
		Tags:     []string{tag3},
		Contacts: []string{uuid.NewV4().String()},
		User:     user1,
		Schedule: moira.ScheduleData{
			StartOffset:    660, // 16:00 (GMT +5) before
			EndOffset:      900, // 20:00 (GMT +5)
			TimezoneOffset: 0,
			Days: []moira.ScheduleDataDay{
				{Enabled: false},
				{Enabled: false},
				{Enabled: true},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
			},
		},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000005",
		Enabled:           false,
		Tags:              []string{tag1, tag2, tag3},
		Contacts:          []string{uuid.NewV4().String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
	{
		ID:                "subscriptionID-00000000000006",
		Enabled:           false,
		Tags:              []string{tag2},
		Contacts:          []string{uuid.NewV4().String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
	{
		ID:                "subscriptionID-00000000000007",
		Enabled:           false,
		Tags:              []string{tag2},
		Contacts:          []string{uuid.NewV4().String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
}
