package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	moira_redis "github.com/moira-alert/moira/database/redis"
)

func updateFrom211(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Update 2.11 -> 2.12 was started")

	ctx := context.Background()
	err := splitNotificationHistoryByContactId(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Update 2.11 -> 2.12 was finished")
	return nil
}

func downgradeTo211(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Downgrade 2.11 -> 2.12 started")

	ctx := context.Background()
	err := unionNotificationHistory(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.11 -> 2.12 was finished")
	return nil
}

var contactNotificationKey = "moira-contact-notifications"

const contactFetchCount = 10_000

func splitNotificationHistoryByContactId(ctx context.Context, logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start splitNotificationHistoryByContactId")

	switch d := database.(type) {
	case *moira_redis.DbConnector:
		client := d.Client()

		events, cursor, err := client.ZScan(ctx, contactNotificationKey, 0, "", contactFetchCount).Result()
		if err != nil {
			return err
		}
		for len(events) > 0 {
			pipe := client.TxPipeline()

			for i := 0; i < len(events); i += 2 {
				notification, err := toNotificationStruct(events[i])
				if err != nil {
					return err
				}

				notificationBytes, err := toNotificationBytes(&notification)
				if err != nil {
					return err
				}

				pipe.ZAdd(
					ctx,
					contactNotificationKey+":"+notification.ContactID,
					&redis.Z{
						Score:  float64(notification.TimeStamp),
						Member: notificationBytes,
					})
			}

			_, err = pipe.Exec(ctx)
			if err != nil {
				return err
			}

			if cursor == 0 {
				break
			}

			events, cursor, err = client.ZScan(ctx, contactNotificationKey, cursor, "", contactFetchCount).Result()
			if err != nil {
				return err
			}
		}

		client.Del(ctx, contactNotificationKey)

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished splitNotificationHistoryByContactId")

	return nil
}

func toNotificationStruct(notificationString string) (moira.NotificationEventHistoryItem, error) {
	var object moira.NotificationEventHistoryItem
	err := json.Unmarshal([]byte(notificationString), &object)
	if err != nil {
		return object, fmt.Errorf("failed to unmarshall event: %s", err.Error())
	}
	return object, nil
}

func toNotificationBytes(notification *moira.NotificationEventHistoryItem) ([]byte, error) {
	bytes, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification event: %s", err.Error())
	}
	return bytes, nil
}

func unionNotificationHistory(ctx context.Context, logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start unionNotificationHistory")

	switch d := database.(type) {
	case *moira_redis.DbConnector:
		client := d.Client()

		contactKeys, err := client.Keys(ctx, contactNotificationKey+":*").Result()
		if err != nil {
			return err
		}

		pipe := client.TxPipeline()

		pipe.ZUnionStore(
			ctx,
			contactNotificationKey,
			&redis.ZStore{
				Keys: append(contactKeys, contactNotificationKey),
			})

		pipe.Del(ctx, contactKeys...)

		_, err = pipe.Exec(ctx)
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished unionNotificationHistory")

	return nil
}
