package controller

import (
	"github.com/go-graphite/carbonapi/date"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/satori/go.uuid"
	"time"
)

func GetUserSubscriptions(database moira.Database, userLogin string) (*dto.SubscriptionList, *api.ErrorResponse) {
	subscriptionIds, err := database.GetUserSubscriptionIds(userLogin)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	subscriptions, err := database.GetSubscriptions(subscriptionIds)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	subscriptionsList := &dto.SubscriptionList{
		List: subscriptions,
	}
	return subscriptionsList, nil
}

func WriteSubscription(database moira.Database, userLogin string, subscription *dto.Subscription) *api.ErrorResponse {
	subscription.User = userLogin
	if subscription.ID != "" {
		data := moira.SubscriptionData(*subscription)
		if err := database.UpdateSubscription(&data); err != nil {
			return api.ErrorInternalServer(err)
		}
	} else {
		subscription.ID = uuid.NewV4().String()
		data := moira.SubscriptionData(*subscription)
		if err := database.CreateSubscription(&data); err != nil {
			return api.ErrorInternalServer(err)
		}
	}
	return nil
}

func DeleteSubscription(database moira.Database, subscriptionId string, userLogin string) *api.ErrorResponse {
	if err := database.DeleteSubscription(subscriptionId, userLogin); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

func SendTestNotification(database moira.Database, subscriptionId string) *api.ErrorResponse {
	var value float64 = 1
	eventData := &moira.EventData{
		SubscriptionID: &subscriptionId,
		Metric:         "Test.metric.value",
		Value:          &value,
		OldState:       "TEST",
		State:          "TEST",
		Timestamp:      int64(date.DateParamToEpoch("now", "", time.Now().Add(-24*time.Hour).Unix(), time.UTC)),
	}

	if err := database.PushEvent(eventData, false); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}
