package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/senders/telegram"
)

var (
	telegramUsersKey = "moira-telegram-users:"
	telegramLockName = "moira-telegram-users:moira-bot-host:"
)

func updateTelegramUsersRecords(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start updateTelegramUsersRecords")

	switch d := database.(type) {
	case *redis.DbConnector:
		pipe := d.Client().TxPipeline()
		iter := d.Client().Scan(d.Context(), 0, telegramUsersKey+"*", 0).Iterator()
		for iter.Next(d.Context()) {
			key := iter.Val()
			if strings.HasPrefix(key, telegramLockName) {
				continue
			}

			oldValue, err := d.Client().Get(d.Context(), key).Result()
			if err != nil {
				return err
			}

			var chatID int64
			var chat *telegram.Chat
			var chatBytes []byte

			chatID, err = strconv.ParseInt(oldValue, 10, 64)
			if err != nil {
				return err
			}

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

			chatBytes, err = json.Marshal(chat)
			if err != nil {
				return err
			}

			pipe.Set(d.Context(), key, string(chatBytes), 0)
		}
		if _, err := pipe.Exec(d.Context()); err != nil {
			return err
		}

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished updateTelegramUsersRecords")

	return nil
}

func downgradeTelegramUsersRecords(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start downgradeTelegramUsersValue")

	switch d := database.(type) {
	case *redis.DbConnector:
		pipe := d.Client().TxPipeline()
		iter := d.Client().Scan(d.Context(), 0, telegramUsersKey+"*", 0).Iterator()
		for iter.Next(d.Context()) {
			key := iter.Val()
			if strings.HasPrefix(key, telegramLockName) {
				continue
			}

			oldValue, err := d.Client().Get(d.Context(), key).Result()
			if err != nil {
				return err
			}

			var newValue string

			chat := telegram.Chat{}
			err = json.Unmarshal([]byte(oldValue), &chat)
			if err != nil {
				return err
			}

			if chat.ID == 0 {
				return fmt.Errorf("chat ID is null")
			} else {
				newValue = strconv.FormatInt(chat.ID, 10)
			}

			pipe.Set(d.Context(), key, newValue, 0)
		}

		if _, err := pipe.Exec(d.Context()); err != nil {
			return err
		}

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished downgradeTelegramUsersValue")

	return nil
}
