package reply

import (
	"github.com/moira-alert/moira-alert"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
)

func Subscription(rep interface{}, err error) (*moira.SubscriptionData, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		return nil, err
	}
	subscription := &moira.SubscriptionData{
		// TODO not sure if this is still necessary, maybe we should just convert database and forget about it
		ThrottlingEnabled: true,
	}
	err = json.Unmarshal(bytes, subscription)
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func Subscriptions(rep interface{}, err error) ([]*moira.SubscriptionData, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		return nil, err
	}
	subscriptions := make([]*moira.SubscriptionData, len(values))
	for i, kk := range values {
		subscription, err2 := Subscription(kk, err)
		if err2 != nil {
			return nil, err2
		}
		subscriptions[i] = subscription
	}
	return subscriptions, nil
}
