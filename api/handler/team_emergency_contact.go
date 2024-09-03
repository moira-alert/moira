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

func teamEmergencyContact(router chi.Router) {
	router.Post("/", createTeamEmergencyContact)
}

// nolint: gofmt,goimports
//
//	@summary	Create team emergency contact
//	@id			create-team-emergency-contact
//	@tags		teamEmergencyContact
//	@accept		json
//	@produce	json
//	@param		teamID	path		string							true	"The ID of team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		emergency-contact	body		dto.EmergencyContact						true	"Emergency contact data"
//	@success	200		{object} dto.SaveEmergencyContactResponse			"Team emergency contact created successfully"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/emergency-contacts [post]
func createTeamEmergencyContact(writer http.ResponseWriter, request *http.Request) {
	emergencyContactDTO := &dto.EmergencyContact{}
	if err := render.Bind(request, emergencyContactDTO); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint:errcheck
		return
	}

	auth := middleware.GetAuth(request)

	response, err := controller.CreateEmergencyContact(database, auth, emergencyContactDTO, "")
	if err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}
