package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

func pattern(router chi.Router) {
	router.Get("/", getAllPatterns)
	router.Delete("/{pattern}", deletePattern)
}

// @summary	Get all patterns
// @id			get-all-patterns
// @tags		pattern
// @produce	json
// @success	200	{object}	dto.PatternList					"Patterns fetched successfully"
// @Failure	422	{object}	api.ErrorRenderExample			"Render error"
// @Failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
// @router		/pattern [get]
func getAllPatterns(writer http.ResponseWriter, request *http.Request) {
	logger := middleware.GetLoggerEntry(request)
	patternsList, err := controller.GetAllPatterns(database, logger)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, patternsList); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

// @summary	Deletes a Moira pattern
// @id			delete-pattern
// @tags		pattern
// @produce	json
// @param		pattern	path	string	true	"Trigger pattern to operate on"	default(DevOps.my_server.hdd.freespace_mbytes)
// @success	200	"Pattern deleted successfully"
// @failure	400	{object}	api.ErrorInvalidRequestExample	"Bad request from client"
// @failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
// @router		/pattern/{pattern} [delete]
func deletePattern(writer http.ResponseWriter, request *http.Request) {
	pattern := chi.URLParam(request, "pattern")
	if pattern == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("pattern must be set"))) //nolint
		return
	}
	err := controller.DeletePattern(database, pattern)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}
