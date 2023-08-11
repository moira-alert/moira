package handler

import (
	"context"
	"errors"
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

// @summary Get all subscriptions
// @id get-user-subscriptions
// @tags subscription
// @produce json
// @success 200 {object} dto.SubscriptionList "Subscriptions fetched successfully"
// @failure 422 {object} api.ErrorRenderExample "Render error"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /subscription [get]
func getUserSubscriptions(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	contacts, err := controller.GetUserSubscriptions(database, userLogin)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, contacts); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// @summary Create a new subscription
// @id create-subscription
// @tags subscription
// @accept json
// @produce json
// @param subscription body dto.Subscription true "Subscription data"
// @success 200 {object} dto.Subscription "Subscription created successfully"
// @failure 400 {object} api.ErrorInvalidRequestExample "Bad request from client"
// @failure 422 {object} api.ErrorRenderExample "Render error"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /subscription [put]
func createSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	userLogin := middleware.GetLogin(request)

	if subscription.AnyTags && len(subscription.Tags) > 0 {
		writer.WriteHeader(http.StatusBadRequest)
		render.Render(writer, request, api.ErrorInvalidRequest( //nolint
			errors.New("if any_tags is true, then the tags must be empty")))
		return
	}
	if err := controller.CreateSubscription(database, userLogin, "", subscription); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
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
			render.Render(writer, request, err) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), subscriptionKey, subscriptionData)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// @summary Update a subscription
// @id update-subscription
// @tags subscription
// @accept json
// @produce json
// @param subscriptionId path string true "ID of the subscription to update" extensions(x-example=bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
// @param subscription body dto.Subscription true "Updated subscription data"
// @success 200 {object} dto.Subscription "Subscription updated successfully"
// @failure 400 {object} api.ErrorInvalidRequestExample "Bad request from client"
// @failure 403 {object} api.ErrorForbiddenExample "Forbidden"
// @failure 404 {object} api.ErrorNotFoundExample "Resource not found"
// @failure 422 {object} api.ErrorRenderExample "Render error"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /subscription/{subscriptionId} [put]
func updateSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		switch err.(type) {
		case dto.ErrProvidedContactsForbidden:
			render.Render(writer, request, api.ErrorForbidden(err.Error())) //nolint
		default:
			render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		}
		return
	}

	if subscription.AnyTags && len(subscription.Tags) > 0 {
		writer.WriteHeader(http.StatusBadRequest)
		render.Render(writer, request, api.ErrorInvalidRequest( //nolint
			errors.New("if any_tags is true, then the tags must be empty")))
		return
	}

	subscriptionData := request.Context().Value(subscriptionKey).(moira.SubscriptionData)

	if err := controller.UpdateSubscription(database, subscriptionData.ID, subscriptionData.User, subscription); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// @summary Delete a subscription
// @id remove-subscription
// @tags subscription
// @produce json
// @param subscriptionId path string true "ID of the subscription to remove" extensions(x-example=bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
// @success 200 "Subscription deleted"
// @failure 403 {object} api.ErrorForbiddenExample "Forbidden"
// @failure 404 {object} api.ErrorNotFoundExample "Resource not found"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /subscription/{subscriptionId} [delete]
func removeSubscription(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := middleware.GetSubscriptionID(request)
	if err := controller.RemoveSubscription(database, subscriptionID); err != nil {
		render.Render(writer, request, err) //nolint
	}
}

// @summary Send a test notification for a subscription
// @id send-test-notification
// @tags subscription
// @produce json
// @param subscriptionId path string true "ID of the subscription to send the test notification" extensions(x-example=bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
// @success 200 "Test notification sent successfully"
// @failure 403 {object} api.ErrorForbiddenExample "Forbidden"
// @failure 404 {object} api.ErrorNotFoundExample "Resource not found"
// @failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @router /subscription/{subscriptionId}/test [put]
func sendTestNotification(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := middleware.GetSubscriptionID(request)
	if err := controller.SendTestNotification(database, subscriptionID); err != nil {
		render.Render(writer, request, err) //nolint
	}
}
