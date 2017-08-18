package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
)

func GetNotifications(database moira.Database, start int64, end int64) (*dto.NotificationsList, *api.ErrorResponse) {
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

func DeleteNotification(database moira.Database, notificationKey string) (*dto.NotificationDeleteResponse, *api.ErrorResponse) {
	result, err := database.RemoveNotification(notificationKey)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	return &dto.NotificationDeleteResponse{Result: result}, nil
}
