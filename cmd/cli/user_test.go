package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/require"
)

const subscriptionPrefix = "subscription_"

func TestUpdateUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "cli", conf.LogPrettyFormat)
	require.NoError(t, err)

	database := redis.NewTestDatabase(logger)
	conf.Cleanup.Whitelist = []string{"Nikolay", ""}

	users := []string{"Aleksey", "Arkadiy", "Emil"}

	require.NoError(t, createDefaultData(database))
	defer func(t *testing.T) {
		t.Helper()
		require.NoError(t, cleanData(database))
	}(t)

	t.Run("Test off notifications", func(t *testing.T) {
		require.NoError(t, usersCleanup(logger, database, users, conf.Cleanup))

		for _, contact := range contacts {
			subscription, err := database.GetSubscription(subscriptionPrefix + contact.ID)
			require.NoError(t, err)

			if strings.Contains(subscription.User, "Another") {
				require.False(t, subscription.Enabled)
			} else {
				require.True(t, subscription.Enabled)
			}
		}
	})

	t.Run("Verify deletion of contacts and subscriptions", func(t *testing.T) {
		conf.Cleanup.Delete = true
		require.NoError(t, usersCleanup(logger, database, users, conf.Cleanup))

		for _, contact := range contacts {
			if !strings.Contains(contact.User, "Another") {
				continue
			}

			_, err := database.GetSubscription(subscriptionPrefix + contact.ID)
			require.Error(t, err)

			_, err = database.GetContact(contact.ID)
			require.Error(t, err)
		}
	})

	t.Run("Test too many users", func(t *testing.T) {
		var manyUsers []string
		for i := 0; i < 100000; i++ {
			manyUsers = append(manyUsers, fmt.Sprintf("User%v", i))
		}

		err := usersCleanup(logger, database, manyUsers, conf.Cleanup)
		require.Equal(t, errors.New("users count is too large"), err)
	})
}

func createDefaultData(database moira.Database) error {
	subscriptions := make([]*moira.SubscriptionData, 0, len(contacts))

	for _, contact := range contacts {
		if err := database.SaveContact(contact); err != nil {
			return err
		}

		subscriptions = append(subscriptions,
			&moira.SubscriptionData{
				ID:       subscriptionPrefix + contact.ID,
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

		if err := database.RemoveSubscription(subscriptionPrefix + contact.ID); err != nil {
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
