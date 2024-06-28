package redis

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira/senders/telegram"
)

const (
	telegramUsersKey   = "moira-telegram-users:"
	telegramLockPrefix = "moira-telegram-users:moira-bot-host:"
)

// UpdateTelegramUsersRecords method that rolls up a 2.11 -> 2.12 migration.
func (connector *DbConnector) UpdateTelegramUsersRecords() error {
	return connector.callFunc(updateTelegramUsersRecordsOnRedisNode)
}

func updateTelegramUsersRecordsOnRedisNode(connector *DbConnector, client redis.UniversalClient) error {
	connector.logger.Info().Msg("Start updateTelegramUsersRecords on redis node")

	ctx := connector.context
	iter := client.Scan(ctx, 0, telegramUsersKey+"*", 0).Iterator()
	pipe := client.TxPipeline()

	for iter.Next(ctx) {
		key := iter.Val()
		if strings.HasPrefix(key, telegramLockPrefix) {
			continue
		}

		oldValue, err := client.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		chatID, err := strconv.ParseInt(oldValue, 10, 64)
		if err != nil {
			connector.logger.Error().
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
			return err
		}

		pipe.Set(ctx, key, string(chatRaw), redis.KeepTTL)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	connector.logger.Info().Msg("Successfully finished updateTelegramUsersRecords on redis node")

	return nil
}

// DowngradeTelegramUsersRecords method that rolls back a 2.11 -> 2.12 migration.
func (connector *DbConnector) DowngradeTelegramUsersRecords() error {
	return connector.callFunc(downgradeTelegramUsersRecordsOnRedisNode)
}

func downgradeTelegramUsersRecordsOnRedisNode(connector *DbConnector, client redis.UniversalClient) error {
	connector.logger.Info().Msg("Start downgradeTelegramUsersValue on redis node")

	ctx := connector.context
	iter := client.Scan(ctx, 0, telegramUsersKey+"*", 0).Iterator()
	pipe := client.TxPipeline()

	for iter.Next(ctx) {
		key := iter.Val()
		if strings.HasPrefix(key, telegramLockPrefix) {
			continue
		}

		oldValue, err := client.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		chat := &telegram.Chat{}
		if err = json.Unmarshal([]byte(oldValue), chat); err != nil {
			connector.logger.Error().
				String("old_value", oldValue).
				Error(err).
				Msg("failed to unmarshal old value chat json")

			continue
		}

		var newValue string
		if chat.ID == 0 {
			connector.logger.Error().
				Msg("chat ID is null")

			continue
		} else {
			newValue = strconv.FormatInt(chat.ID, 10)
		}

		pipe.Set(ctx, key, newValue, redis.KeepTTL)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	connector.logger.Info().Msg("Successfully finished downgradeTelegramUsersValue on redis node")

	return nil
}
