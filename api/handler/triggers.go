package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/index"
	"github.com/moira-alert/moira/remote"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/target"
)

func triggers(cfg *remote.Config, index *index.SearchIndex) func(chi.Router) {
	return func(router chi.Router) {
		router.Use(middleware.RemoteConfigContext(cfg))
		router.Get("/", getAllTriggers)
		router.Put("/", createTrigger)
		router.With(middleware.Paginate(0, 10)).Get("/page", getTriggersPage)
		router.Route("/{triggerId}", trigger)
		router.Route("/search", func(router chi.Router) {
			router.Use(middleware.SearchIndexContext(index))
			router.With(middleware.Paginate(0, 10)).Get("/page", searchTriggersPerPage)
		})
	}
}

func getAllTriggers(writer http.ResponseWriter, request *http.Request) {
	triggersList, errorResponse := controller.GetAllTriggers(database)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func createTrigger(writer http.ResponseWriter, request *http.Request) {
	trigger := &dto.Trigger{}
	if err := render.Bind(request, trigger); err != nil {
		switch err.(type) {
		case target.ErrParseExpr, target.ErrEvalExpr, target.ErrUnknownFunction:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("invalid graphite targets: %s", err.Error())))
		case expression.ErrInvalidExpression:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("invalid expression: %s", err.Error())))
		case remote.ErrRemoteTriggerResponse:
			render.Render(writer, request, api.ErrorRemoteServerUnavailable(err))
		default:
			render.Render(writer, request, api.ErrorInternalServer(err))
		}
		return
	}
	timeSeriesNames := middleware.GetTimeSeriesNames(request)
	response, err := controller.CreateTrigger(database, &trigger.TriggerModel, timeSeriesNames)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func getTriggersPage(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	onlyErrors := getOnlyProblemsFlag(request)
	filterTags := getRequestTags(request)

	page := middleware.GetPage(request)
	size := middleware.GetSize(request)

	triggersList, errorResponse := controller.GetTriggerPage(database, page, size, onlyErrors, filterTags)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func searchTriggersPerPage(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	filterTags := getRequestTags(request)
	searchRequestTextTerms := getSearchRequestTextTerms(request)

	page := middleware.GetPage(request)
	size := middleware.GetSize(request)

	triggersList, errorResponse := controller.FindTriggersPerPage(database, searchIndex, filterTags, searchRequestTextTerms, page, size)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
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

func getSearchRequestTextTerms(request *http.Request) []string {
	searchText := request.FormValue("text")
	searchText, err := url.PathUnescape(searchText)
	if err != nil {
		return []string{}
	}
	if searchText != "" {
		return strings.Fields(searchText)
	}
	return []string{}
}
