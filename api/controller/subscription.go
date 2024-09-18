package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-graphite/carbonapi/date"
	"github.com/gofrs/uuid"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
)

// GetUserSubscriptions get all user subscriptions.
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
		List: make([]moira.SubscriptionData, 0),
	}
	for _, subscription := range subscriptions {
		if subscription != nil {
			subscriptionsList.List = append(subscriptionsList.List, *subscription)
		}
	}
	return subscriptionsList, nil
}

// CreateSubscription create or update subscription.
func CreateSubscription(dataBase moira.Database, auth *api.Authorization, userLogin, teamID string, subscription *dto.Subscription) *api.ErrorResponse {
	if userLogin != "" && teamID != "" {
		return api.ErrorInternalServer(fmt.Errorf("CreateSubscription: cannot create subscription when both userLogin and teamID specified"))
	}
	if subscription.ID == "" {
		uuid4, err := uuid.NewV4()
		if err != nil {
			return api.ErrorInternalServer(err)
		}
		subscription.ID = uuid4.String()
	} else {
		exists, err := isSubscriptionExists(dataBase, subscription.ID)
		if err != nil {
			return api.ErrorInternalServer(err)
		}
		if exists {
			return api.ErrorInvalidRequest(fmt.Errorf("subscription with this ID already exists"))
		}
	}

	// Only admins are allowed to create subscriptions for other users
	if !auth.IsAdmin(userLogin) || subscription.User == "" {
		subscription.User = userLogin
	}
	subscription.TeamID = teamID
	data := moira.SubscriptionData(*subscription)
	if err := dataBase.SaveSubscription(&data); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// GetSubscription returns subscription by it's id.
func GetSubscription(dataBase moira.Database, subscriptionID string) (*dto.Subscription, *api.ErrorResponse) {
	subscription, err := dataBase.GetSubscription(subscriptionID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	dto := dto.Subscription(subscription)
	return &dto, nil
}

// UpdateSubscription updates existing subscription.
func UpdateSubscription(dataBase moira.Database, subscriptionID string, userLogin string, subscription *dto.Subscription) *api.ErrorResponse {
	subscription.ID = subscriptionID
	if subscription.TeamID == "" && subscription.User == "" {
		subscription.User = userLogin
	}

	data := moira.SubscriptionData(*subscription)

	if err := dataBase.SaveSubscription(&data); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// RemoveSubscription deletes subscription.
func RemoveSubscription(database moira.Database, subscriptionID string) *api.ErrorResponse {
	if err := database.RemoveSubscription(subscriptionID); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}

// SendTestNotification push test notification to verify the correct notification settings.
func SendTestNotification(database moira.Database, subscriptionID string) *api.ErrorResponse {
	eventData := &moira.NotificationEvent{
		SubscriptionID: &subscriptionID,
		Metric:         "Test.metric.value",
		Values:         map[string]float64{"t1": 1},
		OldState:       moira.StateTEST,
		State:          moira.StateTEST,
		Timestamp:      date.DateParamToEpoch("now", "", time.Now().Add(-24*time.Hour).Unix(), time.UTC),
	}

	if err := database.PushNotificationEvent(eventData, false); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// CheckUserPermissionsForSubscription checks subscription for existence and permissions for given user.
func CheckUserPermissionsForSubscription(
	dataBase moira.Database,
	subscriptionID string,
	userLogin string,
	auth *api.Authorization,
) (moira.SubscriptionData, *api.ErrorResponse) {
	subscription, err := dataBase.GetSubscription(subscriptionID)
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			return moira.SubscriptionData{}, api.ErrorNotFound(fmt.Sprintf("subscription with ID '%s' does not exists", subscriptionID))
		}
		return moira.SubscriptionData{}, api.ErrorInternalServer(err)
	}

	if auth.IsAdmin(userLogin) {
		return subscription, nil
	}

	if subscription.TeamID != "" {
		teamContainsUser, err := dataBase.IsTeamContainUser(subscription.TeamID, userLogin)
		if err != nil {
			return moira.SubscriptionData{}, api.ErrorInternalServer(err)
		}

		if teamContainsUser {
			return subscription, nil
		}
	}

	if subscription.User == userLogin {
		return subscription, nil
	}

	return moira.SubscriptionData{}, api.ErrorForbidden("you are not permitted")
}

func isSubscriptionExists(dataBase moira.Database, subscriptionID string) (bool, error) {
	_, err := dataBase.GetSubscription(subscriptionID)
	if errors.Is(err, database.ErrNil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
