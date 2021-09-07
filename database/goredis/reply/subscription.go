package reply

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// Subscription converts redis DB reply to moira.SubscriptionData object
func Subscription(rep *redis.StringCmd) (moira.SubscriptionData, error) {
	subscription := moira.SubscriptionData{
		// TODO not sure if this is still necessary, maybe we should just convert database and forget about it
		ThrottlingEnabled: true,
	}
	bytes, err := rep.Bytes()
	if err != nil {
		if err == redis.Nil {
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
