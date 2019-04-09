package redis

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

func TestSubscriptionData(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("SubscriptionData manipulation", t, func(c C) {
		Convey("Save-get-remove subscription", t, func(c C) {
			sub := subscriptions[0]
			Convey("No subscription", t, func(c C) {
				actual, err := dataBase.GetSubscription(sub.ID)
				c.So(err, ShouldBeError)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual, ShouldResemble, moira.SubscriptionData{ThrottlingEnabled: true})
			})
			Convey("Save subscription", t, func(c C) {
				err := dataBase.SaveSubscription(sub)
				c.So(err, ShouldBeNil)
			})
			Convey("Get subscription by id, user and tags", t, func(c C) {
				actual, err := dataBase.GetSubscription(sub.ID)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, *sub)

				actual1, err := dataBase.GetSubscriptions([]string{sub.ID})
				c.So(err, ShouldBeNil)
				c.So(actual1, ShouldResemble, []*moira.SubscriptionData{sub})

				actual2, err := dataBase.GetTagsSubscriptions([]string{tag1})
				c.So(err, ShouldBeNil)
				c.So(actual2, ShouldResemble, []*moira.SubscriptionData{sub})

				actual3, err := dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
				c.So(err, ShouldBeNil)
				c.So(actual3, ShouldResemble, []*moira.SubscriptionData{sub})

				actual4, err := dataBase.GetUserSubscriptionIDs(user1)
				c.So(err, ShouldBeNil)
				c.So(actual4, ShouldResemble, []string{sub.ID})
			})

			Convey("Remove sub", t, func(c C) {
				err := dataBase.RemoveSubscription(sub.ID)
				c.So(err, ShouldBeNil)
			})
			Convey("Get subscription by id, user and tags, should be empty", t, func(c C) {
				actual, err := dataBase.GetSubscription(sub.ID)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual, ShouldResemble, moira.SubscriptionData{ThrottlingEnabled: true})

				actual1, err := dataBase.GetSubscriptions([]string{sub.ID})
				c.So(err, ShouldBeNil)
				c.So(actual1, ShouldResemble, []*moira.SubscriptionData{nil})

				actual3, err := dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
				c.So(err, ShouldBeNil)
				c.So(actual3, ShouldResemble, []*moira.SubscriptionData{})

				actual4, err := dataBase.GetUserSubscriptionIDs(user1)
				c.So(err, ShouldBeNil)
				c.So(actual4, ShouldResemble, []string{})
			})
		})

		Convey("Save batches and remove and check", t, func(c C) {
			ids := make([]string, len(subscriptions))
			for i, sub := range subscriptions {
				ids[i] = sub.ID
			}

			err := dataBase.SaveSubscriptions(subscriptions)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetSubscriptions(ids)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, subscriptions)

			actual1, err := dataBase.GetUserSubscriptionIDs(user1)
			c.So(err, ShouldBeNil)
			c.So(actual1, ShouldHaveLength, len(ids))

			err = dataBase.RemoveSubscription(ids[0])
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetSubscriptions(ids)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldHaveLength, len(ids))

			actual1, err = dataBase.GetUserSubscriptionIDs(user1)
			c.So(err, ShouldBeNil)
			c.So(actual1, ShouldHaveLength, len(ids)-1)
		})

		Convey("Test rewrite subscription", t, func(c C) {
			dataBase.flush()
			sub := *subscriptions[0]

			err := dataBase.SaveSubscription(&sub)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetSubscription(sub.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, sub)

			actual1, err := dataBase.GetUserSubscriptionIDs(user1)
			c.So(err, ShouldBeNil)
			c.So(actual1, ShouldHaveLength, 1)

			sub.User = user2

			err = dataBase.SaveSubscription(&sub)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetSubscription(sub.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, sub)

			actual1, err = dataBase.GetUserSubscriptionIDs(user1)
			c.So(err, ShouldBeNil)
			c.So(actual1, ShouldHaveLength, 0)

			actual1, err = dataBase.GetUserSubscriptionIDs(user2)
			c.So(err, ShouldBeNil)
			c.So(actual1, ShouldHaveLength, 1)

			actual3, err := dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
			c.So(err, ShouldBeNil)
			c.So(actual3, ShouldResemble, []*moira.SubscriptionData{&sub})

			actual4, err := dataBase.GetTagsSubscriptions([]string{tag1, tag3})
			c.So(err, ShouldBeNil)
			c.So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag2})
			c.So(err, ShouldBeNil)
			c.So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})

			sub.Tags = []string{tag1, tag3}

			err = dataBase.SaveSubscription(&sub)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetSubscription(sub.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, sub)

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag1, tag2, tag3})
			c.So(err, ShouldBeNil)
			c.So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag2})
			c.So(err, ShouldBeNil)
			c.So(actual4, ShouldResemble, []*moira.SubscriptionData{})

			actual4, err = dataBase.GetTagsSubscriptions([]string{tag1, tag3})
			c.So(err, ShouldBeNil)
			c.So(actual4, ShouldResemble, []*moira.SubscriptionData{&sub})
		})
	})
}

func TestSubscriptionErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		actual1, err := dataBase.GetSubscription("123")
		c.So(actual1, ShouldResemble, moira.SubscriptionData{ThrottlingEnabled: true})
		c.So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetSubscriptions([]string{"123"})
		c.So(actual2, ShouldBeNil)
		c.So(err, ShouldNotBeNil)

		err = dataBase.SaveSubscriptions(subscriptions)
		c.So(err, ShouldNotBeNil)

		err = dataBase.SaveSubscription(subscriptions[0])
		c.So(err, ShouldNotBeNil)

		err = dataBase.RemoveSubscription(subscriptions[0].ID)
		c.So(err, ShouldNotBeNil)

		actual3, err := dataBase.GetUserSubscriptionIDs("a21213")
		c.So(actual3, ShouldBeNil)
		c.So(err, ShouldNotBeNil)

		actual4, err := dataBase.GetTagsSubscriptions([]string{"123"})
		c.So(actual4, ShouldBeNil)
		c.So(err, ShouldNotBeNil)
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
		Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
	{
		ID:       "subscriptionID-00000000000002",
		Enabled:  true,
		Tags:     []string{tag1},
		Contacts: []string{uuid.Must(uuid.NewV4()).String()},
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
		Contacts: []string{uuid.Must(uuid.NewV4()).String()},
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
		Contacts: []string{uuid.Must(uuid.NewV4()).String()},
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
		Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
	{
		ID:                "subscriptionID-00000000000006",
		Enabled:           false,
		Tags:              []string{tag2},
		Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
	{
		ID:                "subscriptionID-00000000000007",
		Enabled:           false,
		Tags:              []string{tag2},
		Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
		ThrottlingEnabled: true,
		User:              user1,
	},
}
