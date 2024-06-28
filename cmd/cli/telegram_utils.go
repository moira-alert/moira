package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	goredis "github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/senders/telegram"
)

var (
	telegramUsersKey   = "moira-telegram-users:"
	telegramLockPrefix = "moira-telegram-users:moira-bot-host:"
)

// callFunc calls the fn dependent of Redis client type (cluster or standalone).
func callFunc(connector *redis.DbConnector, fn func(connector *redis.DbConnector, client goredis.UniversalClient) error) error {
	client := connector.Client()
	ctx := connector.Context()

	switch c := client.(type) {
	case *goredis.ClusterClient:
		return c.ForEachMaster(ctx, func(ctx context.Context, shard *goredis.Client) error {
			return fn(connector, shard)
		})
	default:
		return fn(connector, client)
	}
}

func updateTelegramUsersRecords(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start updateTelegramUsersRecords")

	switch d := database.(type) {
	case *redis.DbConnector:
		if err := callFunc(d, func(connector *redis.DbConnector, client goredis.UniversalClient) error {
			return updateTelegramUsersRecordsOnRedisNode(connector, client, logger)
		}); err != nil {
			return err
		}

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished updateTelegramUsersRecords")

	return nil
}

func updateTelegramUsersRecordsOnRedisNode(connector *redis.DbConnector, client goredis.UniversalClient, logger moira.Logger) error {
	ctx := connector.Context()
	iter := client.Scan(ctx, 0, telegramUsersKey+"*", 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		if strings.HasPrefix(key, telegramLockPrefix) {
			continue
		}

		oldValue, err := client.Get(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("failed to get value by key: %s, err: %w", key, err)
		}

		chatID, err := strconv.ParseInt(oldValue, 10, 64)
		if err != nil {
			logger.Error().
				String("old_value", oldValue).
				Error(err).
				Msg("failed to parse chatID as int")

			continue
		}

		var chat *telegram.Chat
		if chatID < 0 {
			chat = &telegram.Chat{
				Type: "group",
				ID:   chatID,
			}
		} else {
			chat = &telegram.Chat{
				Type: "private",
				ID:   chatID,
			}
		}

		chatRaw, err := json.Marshal(chat)
		if err != nil {
			return fmt.Errorf("failed to marshal chat: %w", err)
		}

		if err := client.Set(ctx, key, string(chatRaw), goredis.KeepTTL).Err(); err != nil {
			return fmt.Errorf("failed to set %s with value: %s, err: %w", key, string(chatRaw), err)
		}
	}

	return nil
}

func downgradeTelegramUsersRecords(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start downgradeTelegramUsersRecords")

	switch d := database.(type) {
	case *redis.DbConnector:
		if err := callFunc(d, func(connector *redis.DbConnector, client goredis.UniversalClient) error {
			return downgradeTelegramUsersRecordsOnRedisNode(connector, client, logger)
		}); err != nil {
			return err
		}

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished downgradeTelegramUsersRecords")

	return nil
}

func downgradeTelegramUsersRecordsOnRedisNode(connector *redis.DbConnector, client goredis.UniversalClient, logger moira.Logger) error {
	ctx := connector.Context()
	iter := client.Scan(ctx, 0, telegramUsersKey+"*", 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		if strings.HasPrefix(key, telegramLockPrefix) {
			continue
		}

		oldValue, err := client.Get(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("failed to get value by key: %s, err: %w", key, err)
		}

		chat := &telegram.Chat{}
		if err = json.Unmarshal([]byte(oldValue), chat); err != nil {
			logger.Error().
				String("old_value", oldValue).
				Error(err).
				Msg("failed to unmarshal old value chat json")

			continue
		}

		var newValue string
		if chat.ID == 0 {
			logger.Error().
				Msg("chat ID is null")

			continue
		} else {
			newValue = strconv.FormatInt(chat.ID, 10)
		}

		if err := client.Set(ctx, key, newValue, goredis.KeepTTL).Err(); err != nil {
			return fmt.Errorf("failed to set %s with value: %s, err: %w", key, newValue, err)
		}
	}

	return nil
}
