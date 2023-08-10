package handler

import (
	"fmt"
	"net/http"
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

// @summary Gets a paginated list of notifications, all notifications are fetched if end = -1 and start = 0
// @id get-notifications
// @tags notification
// @param start query int false "Default Value: 0" extensions(x-example=1)
// @param end query int false "Default Value: -1" extensions(x-example=15)
// @produce json
// @success 200 {object} dto.NotificationsList "Notifications fetched successfully"
// @failure 400 {object} api.ErrorInvalidRequestExample "Bad request from client"
// @failure 422 {object} api.ErrorRenderExample "Render error"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /notification [get]
func getNotification(writer http.ResponseWriter, request *http.Request) {
	start, err := strconv.ParseInt(request.URL.Query().Get("start"), 10, 64)
	if err != nil {
		start = 0
	}
	end, err := strconv.ParseInt(request.URL.Query().Get("end"), 10, 64)
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

// @summary Delete a notification by id
// @id delete-notification
// @tags notification
// @param id query string true "The ID of updated trigger" extensions(x-example=5A8AF369-86D2-44DD-B514-D47995ED6AF7)
// @produce json
// @success 200 {object} dto.NotificationDeleteResponse "Notification have been deleted"
// @failure 400 {object} api.ErrorInvalidRequestExample "Bad request from client"
// @failure 422 {object} api.ErrorRenderExample "Render error"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /notification [delete]
func deleteNotification(writer http.ResponseWriter, request *http.Request) {
	notificationKey := request.URL.Query().Get("id")
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

// @summary Deletes all available notifications
// @id delete-all-notifications
// @tags notification
// @success 200 "All notifications have been deleted"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /notification/all [delete]
func deleteAllNotifications(writer http.ResponseWriter, request *http.Request) {
	if errorResponse := controller.DeleteAllNotifications(database); errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
	}
}
