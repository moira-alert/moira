package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert"
	moira_middle "github.com/moira-alert/moira-alert/api/middleware"
	"net/http"
)

var database moira.Database

func NewHandler(db moira.Database, log moira.Logger) http.Handler {
	database = db
	router := chi.NewRouter()
	router.Use(moira_middle.Logger(log))
	router.Use(middleware.NoCache)
	router.Use(moira_middle.Recoverer)
	router.Use(render.SetContentType(render.ContentTypeJSON))

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
