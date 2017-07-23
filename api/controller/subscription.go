package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/satori/go.uuid"
	"time"
)

func GetUserSubscriptions(database moira.Database, userLogin string) (*dto.SubscriptionList, *dto.ErrorResponse) {
	subscriptionIds, err := database.GetUserSubscriptionIds(userLogin)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}
	subscriptions, err := database.GetSubscriptions(subscriptionIds)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}
	subscriptionsList := &dto.SubscriptionList{
		List: subscriptions,
	}
	return subscriptionsList, nil
}

func WriteSubscription(database moira.Database, userLogin string, subscription *dto.Subscription) *dto.ErrorResponse {
	subscription.User = userLogin
	if subscription.ID != "" {
		data := moira.SubscriptionData(*subscription)
		if err := database.UpdateSubscription(&data); err != nil {
			return dto.ErrorInternalServer(err)
		}
	} else {
		subscription.ID = uuid.NewV4().String()
		data := moira.SubscriptionData(*subscription)
		if err := database.CreateSubscription(&data); err != nil {
			return dto.ErrorInternalServer(err)
		}
	}
	return nil
}

func DeleteSubscription(database moira.Database, subscriptionId string, userLogin string) *dto.ErrorResponse {
	if err := database.DeleteSubscription(subscriptionId, userLogin); err != nil {
		return dto.ErrorInternalServer(err)
	}
	return nil
}

func SendTestNotification(database moira.Database, subscriptionId string) *dto.ErrorResponse {
	var value float64 = 1
	eventData := &moira.EventData{
		SubscriptionID: &subscriptionId,
		Metric:         "Test.metric.value",
		Value:          &value,
		OldState:       "TEST",
		State:          "TEST",
		Timestamp:      time.Now().Unix(), //todo неправильное время, берется какое-то хитрое графитовское
	}

	if err := database.PushEvent(eventData, false); err != nil {
		return dto.ErrorInternalServer(err)
	}

	return nil
}
