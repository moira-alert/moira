package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/index"
	"github.com/moira-alert/moira/remote"
	"github.com/rs/cors"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	moiramiddle "github.com/moira-alert/moira/api/middleware"
)

var database moira.Database
var searchIndex *index.Index

const contactKey moiramiddle.ContextKey = "contact"
const subscriptionKey moiramiddle.ContextKey = "subscription"

// NewHandler creates new api handler request uris based on github.com/go-chi/chi
func NewHandler(db moira.Database, log moira.Logger, index *index.Index, config *api.Config, remoteConfig *remote.Config, configFile []byte) http.Handler {
	database = db
	searchIndex = index
	router := chi.NewRouter()
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(moiramiddle.UserContext)
	router.Use(moiramiddle.RequestLogger(log))
	router.Use(middleware.NoCache)

	router.NotFound(notFoundHandler)
	router.MethodNotAllowed(methodNotAllowedHandler)

	router.Route("/api", func(router chi.Router) {
		router.Use(moiramiddle.DatabaseContext(database))
		router.Get("/config", webConfig(configFile))
		router.Route("/user", user)
		router.Route("/trigger", triggers(remoteConfig, searchIndex))
		router.Route("/tag", tag)
		router.Route("/pattern", pattern)
		router.Route("/event", event)
		router.Route("/contact", contact)
		router.Route("/subscription", subscription)
		router.Route("/notification", notification)
		router.Route("/health", health)
	})
	if config.EnableCORS {
		return cors.AllowAll().Handler(router)
	}
	return router
}

func webConfig(content []byte) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if content == nil {
			render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("web config file was not loaded")))
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(content)
	})
}

func notFoundHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(404)
	render.Render(writer, request, api.ErrNotFound)
}

func methodNotAllowedHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(405)
	render.Render(writer, request, api.ErrMethodNotAllowed)
}
