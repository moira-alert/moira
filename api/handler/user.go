package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
)

func user(router chi.Router) {
	router.Get("/", getUserName)
	router.Get("/settings", getUserSettings)
}

func getUserName(writer http.ResponseWriter, request *http.Request) {
	userLogin := request.Context().Value("login").(string)
	if err := render.Render(writer, request, &dto.User{Login: userLogin}); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func getUserSettings(writer http.ResponseWriter, request *http.Request) {
	userLogin := request.Context().Value("login").(string)
	userSettings, err := controller.GetUserSettings(database, userLogin)
	if err != nil {
		if err := render.Render(writer, request, err); err != nil {
			render.Render(writer, request, api.ErrorRender(err))
		}
		return
	}

	if err := render.Render(writer, request, userSettings); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}
