package main

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestUpdateTelegramUsersRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, "error", "cli", conf.LogPrettyFormat)
	require.NoError(t, err)

	database := redis.NewTestDatabase(logger, clock.NewSystemClock())
	database.Flush()
	defer database.Flush()

	client := database.Client()
	ctx := database.Context()

	t.Run("Test data migration forwards", func(t *testing.T) {
		t.Run("Given old database", func(t *testing.T) {
			createOldTelegramUserRecords(database)

			t.Run("When migration was applied", func(t *testing.T) {
				err := updateTelegramUsersRecords(logger, database)
				require.NoError(t, err)

				t.Run("Database should be new", func(t *testing.T) {
					result, err := client.Get(ctx, "moira-telegram-users:some telegram group").Result()
					require.NoError(t, err)
					require.Equal(t, `{"chat_id":-1001494975744}`, result)

					result, err = client.Get(ctx, "moira-telegram-users:some telegram group failed migration").Result()
					require.NoError(t, err)
					require.Equal(t, `{"chat_id":-1001494975755}`, result)

					result, err = client.Get(ctx, "moira-telegram-users:@durov").Result()
					require.NoError(t, err)
					require.Equal(t, `{"chat_id":1}`, result)

					result, err = client.Get(ctx, "moira-telegram-users:moira-bot-host:123").Result()
					require.NoError(t, err)
					require.Equal(t, "D4VdnzZDTS/xXF87THARWw==", result)
				})
			})
		})
	})
}

func TestDowngradeTelegramUsersRecords(t *testing.T) {
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

	t.Run("Test data migration backwards", func(t *testing.T) {
		t.Run("Given new database", func(t *testing.T) {
			createNewTelegramUserRecords(database)

			t.Run("When migration was applied", func(t *testing.T) {
				err := downgradeTelegramUsersRecords(logger, database)
				require.NoError(t, err)

				t.Run("Database should be old", func(t *testing.T) {
					result, err := client.Get(ctx, "moira-telegram-users:some telegram group").Result()
					require.NoError(t, err)
					require.Equal(t, "-1001494975744", result)

					result, err = client.Get(ctx, "moira-telegram-users:some telegram group with topic").Result()
					require.NoError(t, err)
					require.Equal(t, `{"chat_id":-1001494975766,"thread_id":1}`, result)

					result, err = client.Get(ctx, "moira-telegram-users:@durov").Result()
					require.NoError(t, err)
					require.Equal(t, "1", result)

					result, err = client.Get(ctx, "moira-telegram-users:@failed_migration").Result()
					require.NoError(t, err)
					require.Equal(t, "2", result)

					result, err = client.Get(ctx, "moira-telegram-users:moira-bot-host:123").Result()
					require.NoError(t, err)
					require.Equal(t, "D4VdnzZDTS/xXF87THARWw==", result)
				})
			})
		})
	})
}

func createOldTelegramUserRecords(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.Set(ctx, "moira-telegram-users:some telegram group", "-1001494975744", goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:some telegram group failed migration", `{"chat_id":-1001494975755}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:@durov", "1", goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:moira-bot-host:123", "D4VdnzZDTS/xXF87THARWw==", goredis.KeepTTL)
	}
}

func createNewTelegramUserRecords(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.Set(ctx, "moira-telegram-users:some telegram group", `{"chat_id":-1001494975744}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:some telegram group with topic", `{"chat_id":-1001494975766,"thread_id":1}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:@durov", `{"chat_id":1}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:@failed_migration", "2", goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:moira-bot-host:123", "D4VdnzZDTS/xXF87THARWw==", goredis.KeepTTL)
	}
}
