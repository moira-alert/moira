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

func createTeamSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint:errcheck
		return
	}
	teamID := middleware.GetTeamID(request)

	if subscription.AnyTags && len(subscription.Tags) > 0 {
		writer.WriteHeader(http.StatusBadRequest)
		render.Render(writer, request, api.ErrorInvalidRequest( //nolint:errcheck
			errors.New("if any_tags is true, then the tags must be empty")))
		return
	}
	if err := controller.CreateSubscription(database, "", teamID, subscription); err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}
	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}
