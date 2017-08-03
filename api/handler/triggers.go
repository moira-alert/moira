package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
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
	var onlyErrors bool
	var filterTags []string

	filterOkCookie, err := request.Cookie("moira_filter_ok")
	if err == http.ErrNoCookie {
		onlyErrors = false
	} else {
		onlyErrors = filterOkCookie.Value == "true"
	}

	fillerTagsCookie, err := request.Cookie("moira_filter_tags")
	if err == http.ErrNoCookie {
		filterTags = make([]string, 0)
	} else {
		str := strings.Split(fillerTagsCookie.Value, ",")
		for _, tag := range str {
			if tag != "" {
				filterTags = append(filterTags, tag)
			}
		}
	}

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
