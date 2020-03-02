package controller

import (
	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	moira2 "github.com/moira-alert/moira/internal/moira"
)

// GetNotifications gets all notifications from current page, if end==-1 && start==0 gets all notifications
func GetNotifications(database moira2.Database, start int64, end int64) (*dto.NotificationsList, *api.ErrorResponse) {
	notifications, total, err := database.GetNotifications(start, end)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	notificationsList := dto.NotificationsList{
		List:  notifications,
		Total: total,
	}
	return &notificationsList, nil
}

// DeleteNotification removes all notifications by notification key
func DeleteNotification(database moira2.Database, notificationKey string) (*dto.NotificationDeleteResponse, *api.ErrorResponse) {
	result, err := database.RemoveNotification(notificationKey)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	return &dto.NotificationDeleteResponse{Result: result}, nil
}

// DeleteAllNotifications removes all notifications
func DeleteAllNotifications(database moira2.Database) *api.ErrorResponse {
	if err := database.RemoveAllNotifications(); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
