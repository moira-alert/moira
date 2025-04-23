package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func health(router chi.Router) {
	router.Get("/notifier", getNotifierState)

	router.With(middleware.AdminOnlyMiddleware()).
		Put("/notifier", setNotifierState)

	router.Get("/system-subscriptions", getSystemSubscriptions)
}

// nolint: gofmt,goimports
//
//	@summary	Get system subscriptions by system tags
//	@id			get-system-subscription
//	@tags		health
//	@produce	json
//	@success	200	{object}	dto.SubscriptionList				"Subscriptions fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/health/system-subscriptions [get]
func getSystemSubscriptions(writer http.ResponseWriter, request *http.Request) {
	const sysTagsKey = "tag"

	sysTags, queryFiltered := request.URL.Query()[sysTagsKey]
	if !queryFiltered {
		checksConfig := middleware.GetSelfStateChecksConfig(request)
		sysTags = checksConfig.GetUniqueSystemTags()
	}

	subs, err := controller.GetSystemSubscriptions(database, sysTags)
	if err != nil {
		_ = render.Render(writer, request, err)
		return
	}

	dto := &dto.SubscriptionList{
		List: make([]moira.SubscriptionData, 0),
	}

	for _, sub := range subs {
		if sub != nil {
			dto.List = append(dto.List, *sub)
		}
	}

	if err := render.Render(writer, request, dto); err != nil {
		_ = render.Render(writer, request, api.ErrorRender(err))
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get notifier state
//	@id			get-notifier-state
//	@tags		health
//	@produce	json
//	@success	200	{object}	dto.NotifierState				"Notifier state retrieved"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/health/notifier [get]
func getNotifierState(writer http.ResponseWriter, request *http.Request) {
	state, err := controller.GetNotifierState(database)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, state); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Set notifier state
//	@id			set-notifier-state
//	@tags		health
//	@produce	json
//	@success	200	{object}	dto.NotifierState				"Notifier state retrieved"
//	@failure	403	{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/health/notifier [put]
func setNotifierState(writer http.ResponseWriter, request *http.Request) {
	state := &dto.NotifierState{}
	if err := render.Bind(request, state); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	if err := controller.UpdateNotifierState(database, state); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, state); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}
