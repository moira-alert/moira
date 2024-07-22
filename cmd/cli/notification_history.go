package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	moira_redis "github.com/moira-alert/moira/database/redis"
)

const (
	contactNotificationKey = "moira-contact-notifications"
)

func splitNotificationHistoryByContactID(ctx context.Context, logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start splitNotificationHistoryByContactID")

	switch d := database.(type) {
	case *moira_redis.DbConnector:
		client := d.Client()
		var splitCount int64

		pipe := client.TxPipeline()

		iterator := client.ZScan(ctx, contactNotificationKey, 0, "", 0).Iterator()
		for iterator.Next(ctx) {
			eventStr := iterator.Val()

			// On 1, 3, 5, ... indexes with have scores, not json
			_, err := strconv.Atoi(eventStr)
			if err == nil {
				continue
			}

			notification, deserializeErr := moira_redis.GetNotificationStruct(eventStr)
			if deserializeErr != nil {
				return fmt.Errorf("failed to deserialize event: %w", deserializeErr)
			}

			notificationBytes, serializeErr := moira_redis.GetNotificationBytes(&notification)
			if serializeErr != nil {
				return fmt.Errorf("failed to serialize event: %w", serializeErr)
			}

			pipe.ZAdd(
				ctx,
				contactNotificationKeyWithID(notification.ContactID),
				&redis.Z{
					Score:  float64(notification.TimeStamp),
					Member: notificationBytes,
				})
			splitCount += 1
		}

		iterErr := iterator.Err()
		if iterErr != nil {
			return fmt.Errorf("error while iterating over notification history: %w", iterErr)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("error while applying changes: %w", err)
		}

		client.Del(ctx, contactNotificationKey)

		logger.Info().
			Int64("split_events", splitCount).
			Msg("Number of contact notifications divided into separate keys")

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished splitNotificationHistoryByContactID")

	return nil
}

func mergeNotificationHistoryOnRedisNode(connector *moira_redis.DbConnector, client redis.UniversalClient, logger moira.Logger) error {
	ctx := connector.Context()
	var contactIDs []string

	iterator := client.Scan(ctx, 0, contactNotificationKeyWithID("*"), 0).Iterator()
	for iterator.Next(ctx) {
		contactIDs = append(contactIDs, iterator.Val())
	}

	iterErr := iterator.Err()
	if iterErr != nil {
		return fmt.Errorf("error while iterating over notification history: %w", iterErr)
	}

	logger.Info().
		Int("contact_ids", len(contactIDs)).
		Msg("Number of contacts in notifications history")

	if len(contactIDs) == 0 {
		return nil
	}

	unionErr := connector.Client().ZUnionStore(
		ctx,
		contactNotificationKey,
		&redis.ZStore{
			Keys: append(contactIDs, contactNotificationKey),
		}).Err()
	if unionErr != nil {
		return fmt.Errorf("error while unioning history: %w", unionErr)
	}

	delErr := client.Del(ctx, contactIDs...).Err()
	if delErr != nil {
		return fmt.Errorf("error while deleting history: %w", delErr)
	}

	return nil
}

func mergeNotificationHistory(ctx context.Context, logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start mergeNotificationHistory")

	switch d := database.(type) {
	case *moira_redis.DbConnector:
		if err := callFunc(d, func(connector *moira_redis.DbConnector, client redis.UniversalClient) error {
			return mergeNotificationHistoryOnRedisNode(connector, client, logger)
		}); err != nil {
			return err
		}

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished mergeNotificationHistory")

	return nil
}

func contactNotificationKeyWithID(contactID string) string {
	return contactNotificationKey + ":" + contactID
}

func handleCleanupNotificationHistoryWithTTL(db moira.Database, ttl int64) error {
	err := db.CleanUpOutdatedNotificationHistory(ttl)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}
