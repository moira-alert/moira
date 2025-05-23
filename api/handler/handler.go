package handler

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	moiramiddle "github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/docs"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/notifier/selfstate"
	"github.com/rs/cors"
)

var (
	database    moira.Database
	searchIndex moira.Searcher
)

const (
	contactKey      moiramiddle.ContextKey = "contact"
	subscriptionKey moiramiddle.ContextKey = "subscription"
)

// NewHandler creates new api handler request uris based on github.com/go-chi/chi.
func NewHandler(
	db moira.Database,
	log moira.Logger,
	index moira.Searcher,
	apiConfig *api.Config,
	metricSourceProvider *metricSource.SourceProvider,
	webConfig *api.WebConfig,
	selfstateConfig *selfstate.ChecksConfig,
) http.Handler {
	database = db
	searchIndex = index

	var contactsTemplate []api.WebContact
	if webConfig != nil {
		contactsTemplate = webConfig.Contacts
	}

	var checksConfig selfstate.ChecksConfig
	if selfstateConfig != nil {
		checksConfig = *selfstateConfig
	}

	contactsTemplateMiddleware := moiramiddle.ContactsTemplateContext(contactsTemplate)

	router := chi.NewRouter()
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(moiramiddle.UserContext)
	router.Use(moiramiddle.RequestLogger(log))
	router.Use(middleware.NoCache)
	router.Use(moiramiddle.LimitsContext(apiConfig.Limits))
	router.Use(moiramiddle.SelfStateChecksContext(checksConfig))

	router.NotFound(notFoundHandler)
	router.MethodNotAllowed(methodNotAllowedHandler)

	//	@title						Moira Alert
	//	@version					master
	//	@externalDocs.description	This is an API description for [Moira Alert API](https://moira.readthedocs.io/en/latest/overview.html)
	//	@externalDocs.description	Check us out on [Github](https://github.com/moira-alert) or look up our [guide](https://moira.readthedocs.io) on getting started with Moira
	//	@externalDocs.url			https://moira.readthedocs.io/en/latest/overview.html
	//	@contact.name				Moira Team
	//	@contact.email				kontur.moira.alert@gmail.com
	//	@license.name				MIT
	//	@license.url				https://opensource.org/licenses/MIT
	//	@license.identifier			MIT
	//	@servers					/api
	//
	//	@tag.name					contact
	//	@tag.description			APIs for working with Moira contacts. For more details, see <https://moira.readthedocs.io/en/latest/installation/webhooks_scripts.html#contact/>
	//
	//	@tag.name					config
	//	@tag.description			View Moira's runtime configuration. For more details, see <https://moira.readthedocs.io/en/latest/installation/configuration.html>
	//
	//	@tag.name					event
	//	@tag.description			APIs for interacting with notification events. See <https://moira.readthedocs.io/en/latest/user_guide/trigger_page.html#event-history/> for details
	//
	//	@tag.name					health
	//	@tag.description			interact with Moira states/health status. See <https://moira.readthedocs.io/en/latest/user_guide/selfstate.html#self-state-monitor/> for details
	//
	//	@tag.name					notification
	//	@tag.description			manage notifications that are currently in queue. See <https://moira.readthedocs.io/en/latest/user_guide/hidden_pages.html#notifications/>
	//
	//	@tag.name					pattern
	//	@tag.description			APIs for interacting with graphite patterns in Moira. See <https://moira.readthedocs.io/en/latest/development/architecture.html#pattern/>
	//
	//	@tag.name					subscription
	//	@tag.description			APIs for managing a user's subscription(s). See <https://moira.readthedocs.io/en/latest/development/architecture.html#subscription/> to learn about Moira subscriptions
	//
	//	@tag.name					tag
	//	@tag.description			APIs for managing tags (a grouping of tags and subscriptions). See <https://moira.readthedocs.io/en/latest/user_guide/subscriptions.html#tags/>
	//
	//	@tag.name					trigger
	//	@tag.description			APIs for interacting with Moira triggers. See <https://moira.readthedocs.io/en/latest/development/architecture.html#trigger/> to learn about Triggers
	//
	//	@tag.name					team
	//	@tag.description			APIs for interacting with Moira teams
	//
	//	@tag.name					teamSubscription
	//	@tag.description			APIs for interacting with Moira subscriptions owned by certain team
	//
	//	@tag.name					teamContact
	//	@tag.description			APIs for interacting with Moira contacts owned by certain team
	//
	//	@tag.name					user
	//	@tag.description			APIs for interacting with Moira users
	router.Route("/api", func(router chi.Router) {
		router.Use(moiramiddle.DatabaseContext(database))
		router.Use(moiramiddle.AuthorizationContext(&apiConfig.Authorization))
		router.Route("/health", health)
		router.Route("/", func(router chi.Router) {
			router.Use(moiramiddle.ReadOnlyMiddleware(apiConfig))
			router.Get("/config", getWebConfig(webConfig))
			router.Route("/user", user)
			router.With(moiramiddle.Triggers(
				apiConfig.MetricsTTL,
			)).Route("/trigger", triggers(metricSourceProvider, searchIndex))
			router.Route("/tag", tag)
			router.Route("/system-tag", systemTag)
			router.Route("/pattern", pattern)
			router.Route("/event", event)
			router.Route("/subscription", subscription)
			router.Route("/notification", notification)
			router.With(contactsTemplateMiddleware).
				Route("/teams", teams)
			router.With(contactsTemplateMiddleware).
				Route("/contact", func(router chi.Router) {
					contact(router)
					contactEvents(router)
				})

			router.Get("/swagger/*", httpSwagger.WrapHandler)
			router.Get("/swagger/doc.json", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(docs.SwaggerInfo.ReadDoc()))
			})
		})
	})

	if apiConfig.EnableCORS {
		return cors.AllowAll().Handler(router)
	}

	return router
}

func notFoundHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusNotFound)
	render.Render(writer, request, api.ErrNotFound) //nolint
}

func methodNotAllowedHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusMethodNotAllowed)
	render.Render(writer, request, api.ErrMethodNotAllowed) //nolint
}
