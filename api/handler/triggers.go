package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/expression"
)

func triggers(metricSourceProvider *metricSource.SourceProvider, searcher moira.Searcher) func(chi.Router) {
	return func(router chi.Router) {
		router.Use(middleware.MetricSourceProvider(metricSourceProvider))
		router.Use(middleware.SearchIndexContext(searcher))
		router.Get("/", getAllTriggers)
		router.Get("/unused", getUnusedTriggers)
		router.Put("/", createTrigger)
		router.Put("/check", triggerCheck)
		router.Route("/{triggerId}", trigger)
		router.With(middleware.Paginate(0, 10)).With(middleware.Pager(false, "")).Get("/search", searchTriggers)
		router.With(middleware.Pager(false, "")).Delete("/search/pager", deletePager)
		// ToDo: DEPRECATED method. Remove in Moira 2.6
		router.With(middleware.Paginate(0, 10)).With(middleware.Pager(false, "")).Get("/page", searchTriggers)
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get all triggers
//	@id			get-all-triggers
//	@tags		trigger
//	@produce	json
//	@success	200	{object}	dto.TriggersList				"Fetched all triggers"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger [get]
func getAllTriggers(writer http.ResponseWriter, request *http.Request) {
	triggersList, errorResponse := controller.GetAllTriggers(database)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get unused triggers
//	@id			get-unused-triggers
//	@tags		trigger
//	@produce	json
//	@success	200	{object}	dto.TriggersList				"Fetched unused triggers"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/unused [get]
func getUnusedTriggers(writer http.ResponseWriter, request *http.Request) {
	triggersList, errorResponse := controller.GetUnusedTriggerIDs(database)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
// createTrigger handler creates moira.Trigger
//
//	@summary	Create a new trigger
//	@id			create-trigger
//	@tags		trigger
//	@accept		json
//	@produce	json
//	@param		validate	query		bool									false	"For validating targets"
//	@param		trigger		body		dto.Trigger								true	"Trigger data"
//	@success	200			{object}	dto.SaveTriggerResponse					"Trigger created successfully"
//	@failure	400			{object}	api.ErrorInvalidRequestExample			"Bad request from client"
//	@failure	422			{object}	api.ErrorRenderExample					"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample			"Internal server error"
//	@failure	503			{object}	api.ErrorRemoteServerUnavailableExample	"Remote server unavailable"
//	@router		/trigger [put]
func createTrigger(writer http.ResponseWriter, request *http.Request) {
	trigger, err := getTriggerFromRequest(request)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	var problems []dto.TreeOfProblems
	if needValidate(request) {
		problems, err = validateTargets(request, trigger)
		if err != nil {
			render.Render(writer, request, err) //nolint
			return
		}

		if problems != nil && dto.DoesAnyTreeHaveError(problems) {
			writeErrorSaveResponse(writer, request, problems)
			return
		}
	}

	if trigger.Desc != nil {
		err := trigger.PopulatedDescription(moira.NotificationEvents{{}})
		if err != nil {
			render.Render(writer, request, api.ErrorRender(err)) //nolint
			return
		}
	}

	timeSeriesNames := middleware.GetTimeSeriesNames(request)

	response, err := controller.CreateTrigger(database, &trigger.TriggerModel, timeSeriesNames)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if problems != nil {
		response.CheckResult.Targets = problems
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

func getTriggerFromRequest(request *http.Request) (*dto.Trigger, *api.ErrorResponse) {
	trigger := &dto.Trigger{}
	if err := render.Bind(request, trigger); err != nil {
		switch err.(type) { // nolint:errorlint
		case local.ErrParseExpr, local.ErrEvalExpr, local.ErrUnknownFunction:
			return nil, api.ErrorInvalidRequest(fmt.Errorf("invalid graphite targets: %s", err.Error()))
		case expression.ErrInvalidExpression:
			return nil, api.ErrorInvalidRequest(fmt.Errorf("invalid expression: %s", err.Error()))
		case api.ErrInvalidRequestContent:
			return nil, api.ErrorInvalidRequest(err)
		case remote.ErrRemoteTriggerResponse:
			response := api.ErrorRemoteServerUnavailable(err)
			middleware.GetLoggerEntry(request).Error().
				String("status", response.StatusText).
				Error(err).
				Msg("Remote server unavailable")
			return nil, response
		case *json.UnmarshalTypeError:
			return nil, api.ErrorInvalidRequest(fmt.Errorf("invalid payload: %s", err.Error()))
		default:
			return nil, api.ErrorInternalServer(err)
		}
	}
	trigger.UpdatedBy = middleware.GetLogin(request)

	return trigger, nil
}

// getMetricTTLByTrigger gets metric ttl duration time from request context for local or remote trigger.
func getMetricTTLByTrigger(request *http.Request, trigger *dto.Trigger) (time.Duration, error) {
	metricTTLs := middleware.GetMetricTTL(request)
	key := trigger.ClusterKey()

	ttl, ok := metricTTLs[key]
	if !ok {
		return 0, fmt.Errorf("can't get ttl: unknown cluster %s", key.String())
	}

	return ttl, nil
}

// nolint: gofmt,goimports
//
//	@summary	Validates trigger target
//	@id			trigger-check
//	@tags		trigger
//	@accept		json
//	@produce	json
//	@param		trigger	body		dto.Trigger						true	"Trigger data"
//	@success	200		{object}	dto.TriggerCheckResponse		"Validation is done, see response body for validation result"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/check [put]
func triggerCheck(writer http.ResponseWriter, request *http.Request) {
	trigger := &dto.Trigger{}
	response := dto.TriggerCheckResponse{}

	if err := render.Bind(request, trigger); err != nil {
		switch err.(type) { // nolint:errorlint
		case expression.ErrInvalidExpression, local.ErrParseExpr, local.ErrEvalExpr, local.ErrUnknownFunction:
			// TODO write comment, why errors are ignored, it is not obvious.
			// In getTriggerFromRequest these types of errors lead to 400.
		default:
			render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
			return
		}
	}

	ttl, err := getMetricTTLByTrigger(request, trigger)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	if len(trigger.Targets) > 0 {
		var err error
		response.Targets, err = dto.TargetVerification(trigger.Targets, ttl, trigger.TriggerSource)
		if err != nil {
			render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
			return
		}
	}

	render.JSON(writer, request, response)
}

// nolint: gofmt,goimports
//
//	@summary		Search triggers. Replaces the deprecated `page` path
//	@description	You can also add filtering by tags, for this purpose add query parameters tags[0]=test, tags[1]=test1 and so on
//	@description	For example, `/api/trigger/search?tags[0]=test&tags[1]=test1`
//	@id				search-triggers
//	@tags			trigger
//	@produce		json
//	@param			onlyProblems	query		boolean							false	"Only include problems"	default(false)
//	@param			text			query		string							false	"Search text"			default(cpu)
//	@param			p				query		integer							false	"Page number"			default(0)
//	@param			size			query		integer							false	"Page size"				default(10)
//	@param			createPager		query		boolean							false	"Create pager"			default(false)
//	@param			pagerID			query		string							false	"Pager ID"				default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param			createdBy		query		string							false	"Created By"			default(moira.team)
//	@success		200				{object}	dto.TriggersList				"Successfully fetched matching triggers"
//	@failure		400				{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure		404				{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure		422				{object}	api.ErrorRenderExample			"Render error"
//	@failure		500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router			/trigger/search [get]
func searchTriggers(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm() //nolint

	createdBy, ok := getTriggerCreatedBy(request)
	searchOptions := moira.SearchOptions{
		Page:                  middleware.GetPage(request),
		Size:                  middleware.GetSize(request),
		OnlyProblems:          getOnlyProblemsFlag(request),
		Tags:                  getRequestTags(request),
		SearchString:          getSearchRequestString(request),
		CreatedBy:             createdBy,
		NeedSearchByCreatedBy: ok,
		CreatePager:           middleware.GetCreatePager(request),
		PagerID:               middleware.GetPagerID(request),
	}

	triggersList, errorResponse := controller.SearchTriggers(database, searchIndex, searchOptions)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Delete triggers pager
//	@id			delete-pager
//	@tags		trigger
//	@produce	json
//	@param		pagerID	query		string									false	"Pager ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200		{object}	dto.TriggersSearchResultDeleteResponse	"Successfully deleted pager"
//	@failure	404		{object}	api.ErrorNotFoundExample				"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample					"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample			"Internal server error"
//	@router		/trigger/search/pager [delete]
func deletePager(writer http.ResponseWriter, request *http.Request) {
	pagerID := middleware.GetPagerID(request)

	response, errorResponse := controller.DeleteTriggersPager(database, pagerID)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

func getRequestTags(request *http.Request) []string {
	var filterTags []string
	i := 0
	for {
		tag := request.FormValue(fmt.Sprintf("tags[%v]", i))
		if tag == "" {
			break
		}
		filterTags = append(filterTags, tag)
		i++
	}
	return filterTags
}

func getOnlyProblemsFlag(request *http.Request) bool {
	onlyProblemsStr := request.FormValue("onlyProblems")
	if onlyProblemsStr != "" {
		onlyProblems, _ := strconv.ParseBool(onlyProblemsStr)
		return onlyProblems
	}
	return false
}

// Checks if the createdBy field has been set.
// If the field has been set, searches for triggers with a specific author createdBy.
// If the field has not been set, searches for triggers with any author.
func getTriggerCreatedBy(request *http.Request) (string, bool) {
	if createdBy, ok := request.Form["createdBy"]; ok {
		return createdBy[0], true
	}
	return "", false
}

func getSearchRequestString(request *http.Request) string {
	searchText := request.FormValue("text")
	searchText = strings.ToLower(searchText)
	searchText, _ = url.PathUnescape(searchText)
	return searchText
}
