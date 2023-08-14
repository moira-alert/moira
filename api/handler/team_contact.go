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

func teamContact(router chi.Router) {
	router.Post("/", createNewTeamContact)
}

// @summary	Create a new team contact
// @id			create-new-team-contact
// @tags		teamContact
// @accept		json
// @produce	json
// @param x-webauth-user header string false "User session token"
// @param		teamID	path		string							true	"The ID of team"	extensions(x-example=d5d98eb3-ee18-4f75-9364-244f67e23b54)
// @param		contact	body		dto.Contact						true	"Team contact data"
// @success	200		{object}	dto.Contact						"Team contact created successfully"
// @failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
// @failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
// @failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
// @failure	422		{object}	api.ErrorRenderExample			"Render error"
// @failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
// @router		/teams/{teamID}/contacts [post]
func createNewTeamContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint:errcheck
		return
	}
	teamID := middleware.GetTeamID(request)

	if err := controller.CreateContact(database, contact, "", teamID); err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}
