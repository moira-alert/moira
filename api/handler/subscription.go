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
		router.Get("/", getSubscription)
		router.Put("/", updateSubscription)
		router.Delete("/", removeSubscription)
		router.Put("/test", sendTestNotification)
	})
}

// nolint: gofmt,goimports
//
//	@summary	Get all subscriptions
//	@id			get-user-subscriptions
//	@tags		subscription
//	@produce	json
//	@success	200	{object}	dto.SubscriptionList			"Subscriptions fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/subscription [get]
func getUserSubscriptions(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	subscriptions, err := controller.GetUserSubscriptions(database, userLogin)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, subscriptions); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Create a new subscription
//	@id			create-subscription
//	@tags		subscription
//	@accept		json
//	@produce	json
//	@param		subscription	body		dto.Subscription				true	"Subscription data"
//	@success	200				{object}	dto.Subscription				"Subscription created successfully"
//	@failure	400				{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	422				{object}	api.ErrorRenderExample			"Render error"
//	@failure	500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/subscription [put]
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

// subscriptionFilter is middleware for check subscription existence and user permissions.
func subscriptionFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		subscriptionID := middleware.GetSubscriptionID(request)
		userLogin := middleware.GetLogin(request)
		auth := middleware.GetAuth(request)
		subscriptionData, err := controller.CheckUserPermissionsForSubscription(database, subscriptionID, userLogin, auth)
		if err != nil {
			render.Render(writer, request, err) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), subscriptionKey, subscriptionData)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// nolint: gofmt,goimports
//
//	@summary	Get subscription by id
//	@id			get-subscription
//	@tags		subscription
//	@produce	json
//	@param		subscriptionID	path		string							true	"ID of the subscription to get"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200				{object}	dto.Subscription				"Subscription fetched successfully"
//	@failure	403				{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	422				{object}	api.ErrorRenderExample			"Render error"
//	@failure	500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/subscription/{subscriptionID} [get]
func getSubscription(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := middleware.GetSubscriptionID(request)
	subscription, err := controller.GetSubscription(database, subscriptionID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Update a subscription
//	@id			update-subscription
//	@tags		subscription
//	@accept		json
//	@produce	json
//	@param		subscriptionID	path		string							true	"ID of the subscription to update"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		subscription	body		dto.Subscription				true	"Updated subscription data"
//	@success	200				{object}	dto.Subscription				"Subscription updated successfully"
//	@failure	400				{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403				{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404				{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422				{object}	api.ErrorRenderExample			"Render error"
//	@failure	500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/subscription/{subscriptionID} [put]
func updateSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		switch err.(type) { // nolint:errorlint
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

// nolint: gofmt,goimports
//
//	@summary	Delete a subscription
//	@id			remove-subscription
//	@tags		subscription
//	@produce	json
//	@param		subscriptionID	path	string	true	"ID of the subscription to remove"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200				"Subscription deleted"
//	@failure	403				{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404				{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/subscription/{subscriptionID} [delete]
func removeSubscription(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := middleware.GetSubscriptionID(request)
	if err := controller.RemoveSubscription(database, subscriptionID); err != nil {
		render.Render(writer, request, err) //nolint
	}
}

// nolint: gofmt,goimports
//
//	@summary	Send a test notification for a subscription
//	@id			send-test-notification
//	@tags		subscription
//	@produce	json
//	@param		subscriptionID	path	string	true	"ID of the subscription to send the test notification"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200				"Test notification sent successfully"
//	@failure	403				{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404				{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/subscription/{subscriptionID}/test [put]
func sendTestNotification(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := middleware.GetSubscriptionID(request)
	if err := controller.SendTestNotification(database, subscriptionID); err != nil {
		render.Render(writer, request, err) //nolint
	}
}
