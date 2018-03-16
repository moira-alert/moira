package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"net/http"
	"strconv"
)

func notification(router chi.Router) {
	router.Get("/", getNotification)
	router.Delete("/", deleteNotification)
	router.Delete("/all", deleteAllNotifications)
}

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
		render.Render(writer, request, errorResponse)
		return
	}
	if err := render.Render(writer, request, notifications); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deleteNotification(writer http.ResponseWriter, request *http.Request) {
	notificationKey := request.URL.Query().Get("id")
	if notificationKey == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Notification id can not be empty")))
		return
	}

	notifications, errorResponse := controller.DeleteNotification(database, notificationKey)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}
	if err := render.Render(writer, request, notifications); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deleteAllNotifications(writer http.ResponseWriter, request *http.Request) {
	notifications, errorResponse := controller.DeleteAllNotifications(database)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}
	if err := render.Render(writer, request, notifications); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}
