package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	moira_redis "github.com/moira-alert/moira/database/redis"
)

const (
	contactNotificationKey = "moira-contact-notifications"
)

func splitNotificationHistoryByContactId(ctx context.Context, logger moira.Logger, database moira.Database, contactFetchCount int64) error {
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

			// events are like [NotificationHistoryItem, score, NotificationHistoryItem, score, ...]
			// so we need to deserialize only even positions
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
			logger.Info().Int("split", len(events)/2).Msg("splitting events...")

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

func mergeNotificationHistory(ctx context.Context, logger moira.Logger, database moira.Database, fetchCount int64) error {
	logger.Info().Msg("Start mergeNotificationHistory")

	switch d := database.(type) {
	case *moira_redis.DbConnector:
		client := d.Client()

		var contactIDs []string

		contactKeys, cursor, err := client.Scan(ctx, 0, contactNotificationKey+":*", fetchCount).Result()
		if err != nil {
			return err
		}

		for len(contactKeys) > 0 {
			contactIDs = append(contactIDs, contactKeys...)

			if cursor == 0 {
				break
			}

			contactKeys, cursor, err = client.Scan(ctx, cursor, contactNotificationKey+":*", fetchCount).Result()
			if err != nil {
				return err
			}
		}

		logger.Info().Int("contacts", len(contactIDs)).Msg("found contacts with notifications history")

		if len(contactIDs) == 0 {
			return nil
		}

		pipe := client.TxPipeline()

		pipe.ZUnionStore(
			ctx,
			contactNotificationKey,
			&redis.ZStore{
				Keys: append(contactIDs, contactNotificationKey),
			})

		pipe.Del(ctx, contactIDs...)

		_, err = pipe.Exec(ctx)
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished mergeNotificationHistory")

	return nil
}
