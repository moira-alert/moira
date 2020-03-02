package reply

import (
	"encoding/json"
	"fmt"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira/internal/database"
)

// Subscription converts redis DB reply to moira.SubscriptionData object
func Subscription(rep interface{}, err error) (moira2.SubscriptionData, error) {
	subscription := moira2.SubscriptionData{
		// TODO not sure if this is still necessary, maybe we should just convert database and forget about it
		ThrottlingEnabled: true,
	}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return subscription, database.ErrNil
		}
		return subscription, fmt.Errorf("failed to read subscription: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &subscription)
	if err != nil {
		return subscription, fmt.Errorf("failed to parse subscription json %s: %s", string(bytes), err.Error())
	}
	return subscription, nil
}

// Subscriptions converts redis DB reply to moira.SubscriptionData objects array
func Subscriptions(rep interface{}, err error) ([]*moira2.SubscriptionData, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira2.SubscriptionData, 0), nil
		}
		return nil, fmt.Errorf("failed to read subscriptions: %s", err.Error())
	}
	subscriptions := make([]*moira2.SubscriptionData, len(values))
	for i, value := range values {
		subscription, err2 := Subscription(value, err)
		if err2 != nil && err2 != database.ErrNil {
			return nil, err2
		} else if err2 == database.ErrNil {
			subscriptions[i] = nil
		} else {
			subscriptions[i] = &subscription
		}
	}
	return subscriptions, nil
}
