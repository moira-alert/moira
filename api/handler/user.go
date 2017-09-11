package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"net/http"
)

func user(router chi.Router) {
	router.Get("/", getUserName)
	router.Get("/settings", getUserSettings)
}

func getUserName(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	if err := render.Render(writer, request, &dto.User{Login: userLogin}); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func getUserSettings(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	userSettings, err := controller.GetUserSettings(database, userLogin)
	if err != nil {
		if err2 := render.Render(writer, request, err); err != nil {
			render.Render(writer, request, api.ErrorRender(err2))
		}
		return
	}

	if err := render.Render(writer, request, userSettings); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}
