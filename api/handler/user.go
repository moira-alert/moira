package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func user(router chi.Router) {
	router.Get("/", getUserName)
	router.Get("/settings", getUserSettings)
}

// nolint: gofmt,goimports
//
//	@summary	Gets the username of the authenticated user if it is available
//	@id			get-user-name
//	@tags		user
//	@produce	json
//	@success	200	{object}	dto.User				"User name fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample	"Render error"
//	@router		/user [get]
func getUserName(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	auth := middleware.GetAuth(request)
	if err := render.Render(writer, request, &dto.User{
		Login:       userLogin,
		Role:        auth.GetRole(userLogin),
		AuthEnabled: auth.IsEnabled(),
	}); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get the user's contacts and subscriptions
//	@id			get-user-settings
//	@tags		user
//	@produce	json
//	@success	200	{object}	dto.UserSettings				"Settings fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/user/settings [get]
func getUserSettings(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	auth := middleware.GetAuth(request)
	userSettings, err := controller.GetUserSettings(database, userLogin, auth)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, userSettings); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}
