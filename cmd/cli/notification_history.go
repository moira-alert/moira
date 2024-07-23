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

func mergeNotificationHistory(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start mergeNotificationHistory")

	switch d := database.(type) {
	case *moira_redis.DbConnector:
		if err := callFunc(d, func(connector *moira_redis.DbConnector, client redis.UniversalClient) error {
			contactIDs, err := scanContactIDs(connector.Context(), client)
			if err != nil {
				return err
			}

			if len(contactIDs) == 0 {
				return nil
			}

			events, fetchErr := fetchNotificationHistoryFromRedisNode(connector, client, logger, contactIDs)
			if fetchErr != nil {
				return fetchErr
			}

			_, err = d.Client().Pipelined(connector.Context(), func(pipe redis.Pipeliner) error {
				for _, event := range events {
					eventBytes, err := moira_redis.GetNotificationBytes(&event)
					if err != nil {
						return fmt.Errorf("failed to serialize notification event: %w", err)
					}
					pipe.ZAdd(d.Context(), contactNotificationKey, &redis.Z{
						Score:  float64(event.TimeStamp),
						Member: eventBytes,
					})
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to add notification history: %w", err)
			}

			logger.Info().
				Msg("successfully added history")

			var totalDelCount int64

			for _, id := range contactIDs {
				delCount, delErr := client.Del(connector.Context(), id).Result()
				if delErr != nil {
					return fmt.Errorf("failed to delete notification history for contact %s on node: %w", id, delErr)
				}
				totalDelCount += delCount
			}

			logger.Info().
				Int64("delete_count", totalDelCount).
				Msg("Number of deleted notification history events from node")

			return nil
		}); err != nil {
			return err
		}

	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished mergeNotificationHistory")

	return nil
}

func fetchNotificationHistoryFromRedisNode(connector *moira_redis.DbConnector, client redis.UniversalClient, logger moira.Logger, contactIDs []string) ([]moira.NotificationEventHistoryItem, error) {
	ctx := connector.Context()

	logger.Info().
		Int("contact_ids", len(contactIDs)).
		Msg("Number of contacts in notifications history")

	if len(contactIDs) == 0 {
		return make([]moira.NotificationEventHistoryItem, 0), nil
	}

	var eventStrings []string

	for _, id := range contactIDs {
		iterator := client.ZScan(ctx, id, 0, "", 0).Iterator()
		for iterator.Next(ctx) {
			eventStr := iterator.Val()

			// On 1, 3, 5, ... indexes with have scores, not json
			_, err := strconv.Atoi(eventStr)
			if err == nil {
				continue
			}

			eventStrings = append(eventStrings, eventStr)
		}

		if err := iterator.Err(); err != nil {
			return nil, fmt.Errorf("error while iterating over contact with id: %s, error: %w", id, err)
		}
	}

	notificationEvents, err := deserializeEvents(eventStrings)
	if err != nil {
		return nil, err
	}

	return notificationEvents, nil
}

func scanContactIDs(ctx context.Context, client redis.UniversalClient) ([]string, error) {
	var contactIDs []string

	iterator := client.Scan(ctx, 0, contactNotificationKeyWithID("*"), 0).Iterator()
	for iterator.Next(ctx) {
		contactIDs = append(contactIDs, iterator.Val())
	}

	iterErr := iterator.Err()
	if iterErr != nil {
		return nil, fmt.Errorf("error while iterating over notification history: %w", iterErr)
	}

	return contactIDs, nil
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

func deserializeEvents(eventStrings []string) ([]moira.NotificationEventHistoryItem, error) {
	notificationEvents := make([]moira.NotificationEventHistoryItem, 0, len(eventStrings))
	for _, str := range eventStrings {
		notification, err := moira_redis.GetNotificationStruct(str)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize notification events: %w", err)
		}

		notificationEvents = append(notificationEvents, notification)
	}
	return notificationEvents, nil
}
