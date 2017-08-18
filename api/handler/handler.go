package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	moira_middle "github.com/moira-alert/moira-alert/api/middleware"
	"net/http"
)

var database moira.Database

func NewHandler(db moira.Database, log moira.Logger) http.Handler {
	database = db
	router := chi.NewRouter()
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(moira_middle.Logger(log))
	router.Use(middleware.NoCache)
	router.Use(moira_middle.Recoverer)

	router.NotFound(notFoundHandler)
	router.MethodNotAllowed(methodNotAllowed)

	router.Route("/api", func(router chi.Router) {
		router.Use(databaseContext)
		router.Use(userContext)
		router.Route("/user", user)
		router.Route("/trigger", triggers)
		router.Route("/tag", tag)
		router.Route("/pattern", pattern)
		router.Route("/event", event)
		router.Route("/contact", contact)
		router.Route("/subscription", subscription)
		router.Route("/notification", notification)
	})
	return router
}

func notFoundHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(404)
	render.Render(writer, request, api.ErrNotFound)
}

func methodNotAllowed(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(405)
	render.Render(writer, request, api.ErrMethodNotAllowed)
}
