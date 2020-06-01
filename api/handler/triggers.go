package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
		router.Put("/", createTrigger)
		router.Get("/check", triggerCheck)
		router.Route("/{triggerId}", trigger)
		router.With(middleware.Paginate(0, 10)).With(middleware.Pager(false, "")).Get("/search", searchTriggers)
		// ToDo: DEPRECATED method. Remove in Moira 2.6
		router.With(middleware.Paginate(0, 10)).With(middleware.Pager(false, "")).Get("/page", searchTriggers)
	}
}

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

func createTrigger(writer http.ResponseWriter, request *http.Request) {
	trigger, err := getTriggerFromRequest(request)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
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

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

func getTriggerFromRequest(request *http.Request) (*dto.Trigger, *api.ErrorResponse) {
	trigger := &dto.Trigger{}
	if err := render.Bind(request, trigger); err != nil {
		switch err.(type) {
		case local.ErrParseExpr, local.ErrEvalExpr, local.ErrUnknownFunction:
			return nil, api.ErrorInvalidRequest(fmt.Errorf("invalid graphite targets: %s", err.Error()))
		case expression.ErrInvalidExpression:
			return nil, api.ErrorInvalidRequest(fmt.Errorf("invalid expression: %s", err.Error()))
		case api.ErrInvalidRequestContent:
			return nil, api.ErrorInvalidRequest(err)
		case remote.ErrRemoteTriggerResponse:
			response := api.ErrorRemoteServerUnavailable(err)
			middleware.GetLoggerEntry(request).Error("%s : %s : %s", response.StatusText, response.ErrorText, err)
			return nil, response
		default:
			return nil, api.ErrorInternalServer(err)
		}
	}

	return trigger, nil
}

func triggerCheck(writer http.ResponseWriter, request *http.Request) {
	ttl := middleware.GetLocalMetricTTL(request)
	trigger := &dto.Trigger{}
	response := dto.TriggerCheckResponse{}

	if err := render.Bind(request, trigger); err != nil {
		switch err.(type) {
		case expression.ErrInvalidExpression, local.ErrParseExpr, local.ErrEvalExpr, local.ErrUnknownFunction:
		default:
			render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
			return
		}
	}

	if len(trigger.Targets) > 0 {
		response.Targets = dto.TargetVerification(trigger.Targets, ttl, trigger.IsRemote)
	}

	render.JSON(writer, request, response)
}

func searchTriggers(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm() //nolint
	onlyErrors := getOnlyProblemsFlag(request)
	filterTags := getRequestTags(request)
	searchString := getSearchRequestString(request)

	page := middleware.GetPage(request)
	size := middleware.GetSize(request)

	createPager := middleware.GetCreatePager(request)
	pagerID := middleware.GetPagerID(request)

	triggersList, errorResponse := controller.SearchTriggers(database, searchIndex, page, size, onlyErrors, filterTags, searchString, createPager, pagerID)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
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

func getSearchRequestString(request *http.Request) string {
	searchText := request.FormValue("text")
	searchText = strings.ToLower(searchText)
	searchText, _ = url.PathUnescape(searchText)
	return searchText
}
