package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
	"strconv"
	"strings"
)

func triggers(router chi.Router) {
	router.Get("/", getAllTriggers)
	router.Put("/", createTrigger)
	router.With(paginate(0, 10)).Get("/page", getTriggersPage)
	router.Route("/{triggerId}", trigger)
}

func getAllTriggers(writer http.ResponseWriter, request *http.Request) {
	triggersList, errorResponse := controller.GetAllTriggers(database)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func createTrigger(writer http.ResponseWriter, request *http.Request) {
	trigger := &dto.Trigger{}
	if err := render.Bind(request, trigger); err != nil {
		render.Render(writer, request, dto.ErrorInvalidRequest(err))
		return
	}
	response, err := controller.CreateTrigger(database, &trigger.Trigger)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func getTriggersPage(writer http.ResponseWriter, request *http.Request) {
	onlyErrors := getFilterOkFlag(request)
	filterTags := getRequestTags(request)

	page := request.Context().Value("page").(int64)
	size := request.Context().Value("size").(int64)

	triggersList, errorResponse := controller.GetTriggerPage(database, page, size, onlyErrors, filterTags)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func getRequestTags(request *http.Request) []string {
	var filterTags []string
	request.ParseForm()
	i := 0
	for {
		tag := request.FormValue(fmt.Sprintf("tags[%v]", i))
		if tag == "" {
			break
		}
		filterTags = append(filterTags, tag)
		i++
	}

	if len(filterTags) != 0 {
		return filterTags
	}

	fillerTagsCookie, err := request.Cookie("moira_filter_tags")
	if err == http.ErrNoCookie {
		filterTags = make([]string, 0)
	} else {
		str := strings.Split(fillerTagsCookie.Value, "%2C")
		for _, tag := range str {
			if tag != "" {
				filterTags = append(filterTags, tag)
			}
		}
	}
	return filterTags
}

func getFilterOkFlag(request *http.Request) bool {
	onlyProblemsStr := request.FormValue("onlyProblems")
	if onlyProblemsStr != "" {
		onlyProblems, _ := strconv.ParseBool(onlyProblemsStr)
		return onlyProblems
	}

	filterOkCookie, err := request.Cookie("moira_filter_ok")
	if err == http.ErrNoCookie {
		return false
	} else {
		return filterOkCookie.Value == "true"
	}
}
