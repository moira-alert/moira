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

// nolint: gofmt,goimports
//
//	@summary	Create a new team contact
//	@id			create-new-team-contact
//	@tags		teamContact
//	@accept		json
//	@produce	json
//	@param		teamID	path		string							true	"The ID of team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		contact	body		dto.Contact						true	"Team contact data"
//	@success	200		{object}	dto.Contact						"Team contact created successfully"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/contacts [post]
func createNewTeamContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint:errcheck
		return
	}
	teamID := middleware.GetTeamID(request)
	auth := middleware.GetAuth(request)

	if err := controller.CreateContact(database, auth, contact, "", teamID); err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}
