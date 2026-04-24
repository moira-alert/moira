package main

import (
	"testing"

	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	"github.com/moira-alert/moira"

	rds "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, "error", "cli", conf.LogPrettyFormat)
	require.NoError(t, err)

	database := redis.NewTestDatabase(logger)

	database.Flush()
	defer database.Flush()

	client := database.Client()
	ctx := database.Context()

	t.Run("Test data migration forwards", func(t *testing.T) {
		t.Run("Given old database", func(t *testing.T) {
			createDataWithOldKeys(database)

			valueStoredAtKey := client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
			require.Empty(t, valueStoredAtKey)

			valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
			require.Empty(t, valueStoredAtKey)

			valueStoredAtKey = client.SMembers(ctx, "moira-remote-triggers-list").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Val()
			require.Empty(t, valueStoredAtKey)

			valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
			require.Empty(t, valueStoredAtKey)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
			require.Empty(t, valueStoredAtKey)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
			require.Empty(t, valueStoredAtKey)

			valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
			require.Empty(t, valueStoredAtKey)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
			require.Empty(t, valueStoredAtKey)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
			require.Len(t, valueStoredAtKey, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
			require.Empty(t, valueStoredAtKey)

			t.Run("When migration was applied", func(t *testing.T) {
				err := addRedisClusterSupport(logger, database)
				require.NoError(t, err)

				t.Run("Database should be new", func(t *testing.T) {
					valueStoredAtKey = client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
					require.Len(t, valueStoredAtKey, 3)

					valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
					require.Len(t, valueStoredAtKey, 3)

					valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
					require.Len(t, valueStoredAtKey, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
					require.Len(t, valueStoredAtKey, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
					require.Len(t, valueStoredAtKey, 3)

					valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
					require.Len(t, valueStoredAtKey, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
					require.Len(t, valueStoredAtKey, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
					require.Empty(t, valueStoredAtKey)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
					require.Len(t, valueStoredAtKey, 3)
				})
			})
		})

		t.Run("Test data migration reverse", func(t *testing.T) {
			t.Run("Given new database", func(t *testing.T) {
				createDataWithNewKeys(database)

				valueStoredAtKey := client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
				require.Empty(t, valueStoredAtKey)

				valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
				require.Empty(t, valueStoredAtKey)

				valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-remote-triggers-list").Val()
				require.Empty(t, valueStoredAtKey)

				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
				require.Empty(t, valueStoredAtKey)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
				require.Empty(t, valueStoredAtKey)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
				require.Empty(t, valueStoredAtKey)

				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
				require.Empty(t, valueStoredAtKey)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
				require.Empty(t, valueStoredAtKey)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
				require.Len(t, valueStoredAtKey, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
				require.Empty(t, valueStoredAtKey)

				t.Run("When migration was reversed", func(t *testing.T) {
					err := removeRedisClusterSupport(logger, database)
					require.NoError(t, err)

					t.Run("Database should be old", func(t *testing.T) {
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
						require.Len(t, valueStoredAtKey, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
						require.Len(t, valueStoredAtKey, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-remote-triggers-list").Val()
						require.Len(t, valueStoredAtKey, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
						require.Len(t, valueStoredAtKey, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
						require.Len(t, valueStoredAtKey, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
						require.Len(t, valueStoredAtKey, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
						require.Len(t, valueStoredAtKey, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
						require.Len(t, valueStoredAtKey, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
						require.Empty(t, valueStoredAtKey)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
						require.Len(t, valueStoredAtKey, 3)
					})
				})
			})
		})
	})
}

func Test_renameKey(t *testing.T) {
	logger, _ := logging.GetLogger("Test Worker")
	oldKey := "my_test_key"
	newKey := "my_new_test_key"

	t.Run("Something was renamed", func(t *testing.T) {
		database := redis.NewTestDatabase(logger)

		database.Flush()
		defer database.Flush()

		err := database.Client().Set(database.Context(), oldKey, "123", 0).Err()
		require.NoError(t, err)

		err = renameKey(database, oldKey, newKey)
		require.NoError(t, err)

		res, err := database.Client().Get(database.Context(), newKey).Result()
		require.NoError(t, err)
		require.Equal(t, "123", res)

		err = database.Client().Get(database.Context(), oldKey).Err()
		require.Equal(t, rds.Nil, err)
	})

	t.Run("Nothing was renamed", func(t *testing.T) {
		database := redis.NewTestDatabase(logger)

		database.Flush()
		defer database.Flush()

		err := database.Client().Set(database.Context(), oldKey, "123", 0).Err()
		require.NoError(t, err)

		err = renameKey(database, "no_exist_key", newKey)
		require.NoError(t, err)

		err = database.Client().Get(database.Context(), newKey).Err()
		require.Equal(t, rds.Nil, err)

		res, err := database.Client().Get(database.Context(), oldKey).Result()
		require.NoError(t, err)
		require.Equal(t, "123", res)
	})
}

func Test_changeKeysPrefix(t *testing.T) {
	logger, _ := logging.GetLogger("Test Worker")
	oldKey := "my_test_key"
	newKey := "my_new_test_key"

	t.Run("Something was renamed", func(t *testing.T) {
		database := redis.NewTestDatabase(logger)

		database.Flush()
		defer database.Flush()

		err := database.Client().Set(database.Context(), oldKey+"1", "1", 0).Err()
		require.NoError(t, err)
		err = database.Client().Set(database.Context(), oldKey+"2", "2", 0).Err()
		require.NoError(t, err)
		err = database.Client().Set(database.Context(), oldKey+"3", "3", 0).Err()
		require.NoError(t, err)

		err = changeKeysPrefix(database, oldKey, newKey)
		require.NoError(t, err)

		res, err := database.Client().Get(database.Context(), newKey+"1").Result()
		require.NoError(t, err)
		require.Equal(t, "1", res)
		res, err = database.Client().Get(database.Context(), newKey+"2").Result()
		require.NoError(t, err)
		require.Equal(t, "2", res)
		res, err = database.Client().Get(database.Context(), newKey+"3").Result()
		require.NoError(t, err)
		require.Equal(t, "3", res)

		err = database.Client().Get(database.Context(), oldKey+"1").Err()
		require.Equal(t, rds.Nil, err)
		err = database.Client().Get(database.Context(), oldKey+"2").Err()
		require.Equal(t, rds.Nil, err)
		err = database.Client().Get(database.Context(), oldKey+"3").Err()
		require.Equal(t, rds.Nil, err)
	})

	t.Run("Nothing was renamed", func(t *testing.T) {
		database := redis.NewTestDatabase(logger)

		database.Flush()
		defer database.Flush()

		err := database.Client().Set(database.Context(), oldKey+"1", "1", 0).Err()
		require.NoError(t, err)
		err = database.Client().Set(database.Context(), oldKey+"2", "2", 0).Err()
		require.NoError(t, err)
		err = database.Client().Set(database.Context(), oldKey+"3", "3", 0).Err()
		require.NoError(t, err)

		err = renameKey(database, "no_exist_key", newKey)
		require.NoError(t, err)

		err = database.Client().Get(database.Context(), newKey).Err()
		require.Equal(t, rds.Nil, err)

		res, err := database.Client().Get(database.Context(), oldKey+"1").Result()
		require.NoError(t, err)
		require.Equal(t, "1", res)
		res, err = database.Client().Get(database.Context(), oldKey+"2").Result()
		require.NoError(t, err)
		require.Equal(t, "2", res)
		res, err = database.Client().Get(database.Context(), oldKey+"3").Result()
		require.NoError(t, err)
		require.Equal(t, "3", res)
	})
}

func createDataWithOldKeys(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.SAdd(ctx, "moira-any-tags-subscriptions", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-any-tags-subscriptions", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-any-tags-subscriptions", "subscriptionID-00000000000003")

		client.SAdd(ctx, "moira-triggers-list", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-triggers-list", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-triggers-list", "triggerID-0000000000003")

		client.SAdd(ctx, "moira-remote-triggers-list", "triggerID-0000000000004")
		client.SAdd(ctx, "moira-remote-triggers-list", "triggerID-0000000000005")
		client.SAdd(ctx, "moira-remote-triggers-list", "triggerID-0000000000006")

		client.SAdd(ctx, "moira-tag-subscriptions:tag1", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-tag-subscriptions:tag1", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-tag-subscriptions:tag1", "subscriptionID-00000000000003")
		client.SAdd(ctx, "moira-tag-subscriptions:tag2", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-tag-subscriptions:tag2", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-tag-subscriptions:tag2", "subscriptionID-00000000000003")
		client.SAdd(ctx, "moira-tag-subscriptions:tag3", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-tag-subscriptions:tag3", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-tag-subscriptions:tag3", "subscriptionID-00000000000003")

		client.SAdd(ctx, "moira-tag-triggers:tag1", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-tag-triggers:tag1", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-tag-triggers:tag1", "triggerID-0000000000003")
		client.SAdd(ctx, "moira-tag-triggers:tag2", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-tag-triggers:tag2", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-tag-triggers:tag2", "triggerID-0000000000003")
		client.SAdd(ctx, "moira-tag-triggers:tag3", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-tag-triggers:tag3", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-tag-triggers:tag3", "triggerID-0000000000003")
	}
}

func createDataWithNewKeys(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.SAdd(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions", "subscriptionID-00000000000003")

		client.SAdd(ctx, "{moira-triggers-list}:moira-triggers-list", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-triggers-list}:moira-triggers-list", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-triggers-list}:moira-triggers-list", "triggerID-0000000000003")

		client.SAdd(ctx, "{moira-triggers-list}:moira-remote-triggers-list", "triggerID-0000000000004")
		client.SAdd(ctx, "{moira-triggers-list}:moira-remote-triggers-list", "triggerID-0000000000005")
		client.SAdd(ctx, "{moira-triggers-list}:moira-remote-triggers-list", "triggerID-0000000000006")

		client.SAdd(ctx, "{moira-tag-subscriptions}:tag1", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag1", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag1", "subscriptionID-00000000000003")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag2", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag2", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag2", "subscriptionID-00000000000003")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag3", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag3", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag3", "subscriptionID-00000000000003")

		client.SAdd(ctx, "{moira-tag-triggers}:tag1", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-tag-triggers}:tag1", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-tag-triggers}:tag1", "triggerID-0000000000003")
		client.SAdd(ctx, "{moira-tag-triggers}:tag2", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-tag-triggers}:tag2", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-tag-triggers}:tag2", "triggerID-0000000000003")
		client.SAdd(ctx, "{moira-tag-triggers}:tag3", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-tag-triggers}:tag3", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-tag-triggers}:tag3", "triggerID-0000000000003")
	}
}
