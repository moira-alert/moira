package main

import (
	"strings"
	"testing"

	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"

	"github.com/moira-alert/moira"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUpdateUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	conf := getDefault()
	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "cli")
	if err != nil {
		t.Fatal(err)
	}

	database := redis.NewDatabase(logger, conf.Redis.GetSettings(), redis.Cli)
	conf.Cleanup.Whitelist = []string{"Nikolay", ""}

	users := []string{"Aleksey", "Arkadiy", "Emil"}

	Convey("Test clean users in moira", t, func() {
		if err := createDefaultData(database); err != nil {
			t.Fatal(err)
		}

		defer func(t *testing.T) {
			if err := cleanData(database); err != nil {
				t.Fatal(err)
			}
		}(t)

		Convey("Test off notifications", func() {
			So(usersCleanup(logger, database, users, conf.Cleanup), ShouldBeNil)
			for _, contact := range contacts {
				subscription, err := database.GetSubscription("subscription_" + contact.ID)

				So(err, ShouldBeNil)

				if strings.Contains(subscription.User, "Another") {
					So(subscription.Enabled, ShouldBeFalse)
				} else {
					So(subscription.Enabled, ShouldBeTrue)
				}
			}
		})

		Convey("Verify deletion of contacts and subscriptions", func() {
			conf.Cleanup.Delete = true
			So(usersCleanup(logger, database, users, conf.Cleanup), ShouldBeNil)
			for _, contact := range contacts {
				if !strings.Contains(contact.User, "Another") {
					continue
				}

				_, err := database.GetSubscription("subscription_" + contact.ID)
				So(err, ShouldNotBeNil)

				_, err = database.GetContact(contact.ID)
				So(err, ShouldNotBeNil)
			}
		})
	})

}

func createDefaultData(database moira.Database) error {
	subscriptions := make([]*moira.SubscriptionData, 0, len(contacts))

	for _, contact := range contacts {
		if err := database.SaveContact(contact); err != nil {
			return err
		}

		subscriptions = append(subscriptions,
			&moira.SubscriptionData{ID: "subscription_" + contact.ID,
				User:     contact.User,
				Enabled:  true,
				Tags:     []string{"Tag" + contact.User},
				Contacts: []string{contact.ID},
			},
		)
	}

	if err := database.SaveSubscriptions(subscriptions); err != nil {
		return err
	}

	return nil
}

func cleanData(database moira.Database) error {

	for _, contact := range contacts {
		if err := database.RemoveContact(contact.ID); err != nil {
			return err
		}

		if err := database.RemoveSubscription("subscription_" + contact.ID); err != nil {
			return err
		}
	}

	return nil
}

var contacts = []*moira.ContactData{
	{ID: "cli_id_00000000000001", User: "Aleksey"},
	{ID: "cli_id_00000000000002", User: "Arkadiy"},
	{ID: "cli_id_00000000000003", User: "Emil"},
	{ID: "cli_id_00000000000004", User: "Nikolay"},
	{ID: "cli_id_00000000000005", User: "Another1"},
	{ID: "cli_id_00000000000006", User: "Another2"},
	{ID: "cli_id_00000000000007", User: ""},
}
