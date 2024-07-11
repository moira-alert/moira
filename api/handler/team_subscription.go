package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func teamSubscription(router chi.Router) {
	router.Post("/", createTeamSubscription)
}

// nolint: gofmt,goimports
//
//	@summary	Create a new team subscription
//	@id			create-new-team-subscription
//	@tags		teamSubscription
//	@accept		json
//	@produce	json
//	@param		teamID			path		string							true	"The ID of team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		subscription	body		dto.Subscription				true	"Team subscription data"
//	@success	200				{object}	dto.Subscription				"Team subscription created successfully"
//	@failure	400				{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403				{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404				{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422				{object}	api.ErrorRenderExample			"Render error"
//	@failure	500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/subscriptions [post]
func createTeamSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint:errcheck
		return
	}
	teamID := middleware.GetTeamID(request)
	auth := middleware.GetAuth(request)

	if subscription.AnyTags && len(subscription.Tags) > 0 {
		writer.WriteHeader(http.StatusBadRequest)
		render.Render(writer, request, api.ErrorInvalidRequest( //nolint:errcheck
			errors.New("if any_tags is true, then the tags must be empty")))
		return
	}
	if err := controller.CreateSubscription(database, auth, "", teamID, subscription); err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}
	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}
