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
