package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert"
	"net/http"
)

var database moira.Database
var logger moira.Logger

func NewHandler(db moira.Database, logger moira.Logger) http.Handler {
	database = db
	logger = logger
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.NoCache) //todo неадекватно много всего проставляет, разобраться
	router.Use(middleware.Recoverer)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	router.Route("/api", func(router chi.Router) {
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
