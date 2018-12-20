package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func subscription(router chi.Router) {
	router.Get("/", getUserSubscriptions)
	router.Put("/", createSubscription)
	router.Route("/{subscriptionId}", func(router chi.Router) {
		router.Use(middleware.SubscriptionContext)
		router.Use(subscriptionFilter)
		router.Put("/", updateSubscription)
		router.Delete("/", removeSubscription)
		router.Put("/test", sendTestNotification)
	})
}

func getUserSubscriptions(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	contacts, err := controller.GetUserSubscriptions(database, userLogin)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, contacts); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func createSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	userLogin := middleware.GetLogin(request)

	if err := controller.CreateSubscription(database, userLogin, subscription); err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

// subscriptionFilter is middleware for check subscription existence and user permissions
func subscriptionFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contactID := middleware.GetSubscriptionID(request)
		userLogin := middleware.GetLogin(request)
		subscriptionData, err := controller.CheckUserPermissionsForSubscription(database, contactID, userLogin)
		if err != nil {
			render.Render(writer, request, err)
			return
		}
		ctx := context.WithValue(request.Context(), subscriptionKey, subscriptionData)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func updateSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		switch err.(type) {
		case dto.SubscriptionHasAnotherUserContact:
			render.Render(writer, request, api.ErrorForbidden(err.Error()))
		default:
			render.Render(writer, request, api.ErrorInvalidRequest(err))
		}
		return
	}
	subscriptionData := request.Context().Value(subscriptionKey).(moira.SubscriptionData)

	if err := controller.UpdateSubscription(database, subscriptionData.ID, subscriptionData.User, subscription); err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func removeSubscription(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := middleware.GetSubscriptionID(request)
	if err := controller.RemoveSubscription(database, subscriptionID); err != nil {
		render.Render(writer, request, err)
	}
}

func sendTestNotification(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := middleware.GetSubscriptionID(request)
	if err := controller.SendTestNotification(database, subscriptionID); err != nil {
		render.Render(writer, request, err)
	}
}
