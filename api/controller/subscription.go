package controller

import (
	"time"

	"github.com/go-graphite/carbonapi/date"
	"github.com/satori/go.uuid"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
)

// GetUserSubscriptions get all user subscriptions
func GetUserSubscriptions(database moira.Database, userLogin string) (*dto.SubscriptionList, *api.ErrorResponse) {
	subscriptionIDs, err := database.GetUserSubscriptionIDs(userLogin)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	subscriptions, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	subscriptionsList := &dto.SubscriptionList{
		List: subscriptions,
	}
	return subscriptionsList, nil
}

// WriteSubscription create or update subscription
func WriteSubscription(database moira.Database, userLogin string, subscription *dto.Subscription) *api.ErrorResponse {
	subscription.User = userLogin
	if subscription.ID == "" {
		subscription.ID = uuid.NewV4().String()
	}
	data := moira.SubscriptionData(*subscription)
	if err := database.SaveSubscription(&data); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// RemoveSubscription deletes subscription
func RemoveSubscription(database moira.Database, subscriptionID string, userLogin string) *api.ErrorResponse {
	if err := database.RemoveSubscription(subscriptionID, userLogin); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// SendTestNotification push test notification to verify the correct notification settings
func SendTestNotification(database moira.Database, subscriptionID string) *api.ErrorResponse {
	var value float64 = 1
	eventData := &moira.NotificationEvent{
		SubscriptionID: &subscriptionID,
		Metric:         "Test.metric.value",
		Value:          &value,
		OldState:       "TEST",
		State:          "TEST",
		Timestamp:      int64(date.DateParamToEpoch("now", "", time.Now().Add(-24*time.Hour).Unix(), time.UTC)),
	}

	if err := database.PushNotificationEvent(eventData, false); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}
