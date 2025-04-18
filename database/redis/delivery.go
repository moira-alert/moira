package redis

import (
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira/database"
)

// AddDeliveryChecksData adds given data to sorted by timestamp set relative to contact type.
func (connector *DbConnector) AddDeliveryChecksData(contactType string, timestamp int64, data string) error {
	client := connector.Client()
	ctx := connector.Context()

	return client.ZAdd(
		ctx,
		deliveryCheckKeyWithContactType(contactType),
		&redis.Z{
			Score:  float64(timestamp),
			Member: data,
		}).Err()
}

// GetDeliveryChecksData reads data from for given tim range relative to contact type.
func (connector *DbConnector) GetDeliveryChecksData(contactType string, from string, to string) ([]string, error) {
	client := connector.Client()
	ctx := connector.Context()

	res, err := client.ZRangeByScore(
		ctx,
		deliveryCheckKeyWithContactType(contactType),
		&redis.ZRangeBy{
			Min:    from,
			Max:    to,
			Offset: 0,
			Count:  -1,
		}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, database.ErrNil
		}

		return nil, err
	}

	return res, nil
}

// RemoveDeliveryChecksData removes data from for given time range relative to contact type.
func (connector *DbConnector) RemoveDeliveryChecksData(contactType string, from string, to string) (int64, error) {
	client := connector.Client()
	ctx := connector.Context()

	return client.ZRemRangeByScore(ctx, deliveryCheckKeyWithContactType(contactType), from, to).Result()
}

const deliveryCheckKey = "moira-delivery-check"

func deliveryCheckKeyWithContactType(contactType string) string {
	return deliveryCheckKey + ":" + contactType
}
