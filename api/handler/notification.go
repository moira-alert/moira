package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
)

func notification(router chi.Router) {
	router.Get("/", getNotification)
	router.Delete("/", deleteNotification)
	router.Delete("/all", deleteAllNotifications)
}

// nolint: gofmt,goimports
//
//	@summary	Gets a paginated list of notifications, all notifications are fetched if end = -1 and start = 0
//	@id			get-notifications
//	@tags		notification
//	@produce	json
//	@param		start	query		int								false	"Default Value: 0"	default(0)
//	@param		end		query		int								false	"Default Value: -1"	default(-1)
//	@success	200		{object}	dto.NotificationsList			"Notifications fetched successfully"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/notification [get]
func getNotification(writer http.ResponseWriter, request *http.Request) {
	urlValues, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	start, err := strconv.ParseInt(urlValues.Get("start"), 10, 64)
	if err != nil {
		start = 0
	}

	end, err := strconv.ParseInt(urlValues.Get("end"), 10, 64)
	if err != nil {
		end = -1
	}

	notifications, errorResponse := controller.GetNotifications(database, start, end)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
		return
	}

	if err := render.Render(writer, request, notifications); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

// nolint: gofmt,goimports
//
//	@summary	Delete a notification by id
//	@id			delete-notification
//	@tags		notification
//	@param		id	query	string	true	"The ID of deleted notification"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@produce	json
//	@success	200	{object}	dto.NotificationDeleteResponse	"Notification have been deleted"
//	@failure	400	{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/notification [delete]
func deleteNotification(writer http.ResponseWriter, request *http.Request) {
	urlValues, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	notificationKey := urlValues.Get("id")
	if notificationKey == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("notification id can not be empty"))) //nolint
		return
	}

	notifications, errorResponse := controller.DeleteNotification(database, notificationKey)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
		return
	}

	if err := render.Render(writer, request, notifications); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

func deleteAllNotifications(writer http.ResponseWriter, request *http.Request) {
	if errorResponse := controller.DeleteAllNotifications(database); errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
	}
}
