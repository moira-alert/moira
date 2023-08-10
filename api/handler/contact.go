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

func contact(router chi.Router) {
	router.Get("/", getAllContacts)
	router.Put("/", createNewContact)
	router.Route("/{contactId}", func(router chi.Router) {
		router.Use(middleware.ContactContext)
		router.Use(contactFilter)
		router.Put("/", updateContact)
		router.Delete("/", removeContact)
		router.Post("/test", sendTestContactNotification)
	})
}

// @Summary Gets all Moira contacts
// @ID get-all-contacts
// @Produce json
// @Success 200 {object} dto.ContactList "Contacts fetched successfully"
// @Failure 422 {object} api.ErrorRenderExample "Render error"
// @Router /contact [get]
// @Tags contact
func getAllContacts(writer http.ResponseWriter, request *http.Request) {
	contacts, err := controller.GetAllContacts(database)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, contacts); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// @Summary Creates a new contact notification for the current user
// @ID create-new-contact
// @Accept json
// @Produce json
// @Param contact body dto.Contact true "Contact data"
// @Success 200 {object} dto.Contact "Contact created successfully"
// @Failure 400 {object} api.ErrorInvalidRequestExample "Bad request from client"
// @Failure 422 {object} api.ErrorRenderExample "Render error"
// @Failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @Router /contact [put]
// @Tags contact
func createNewContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	userLogin := middleware.GetLogin(request)

	if err := controller.CreateContact(database, contact, userLogin, ""); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// contactFilter is middleware for check contact existence and user permissions
func contactFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contactID := middleware.GetContactID(request)
		userLogin := middleware.GetLogin(request)
		contactData, err := controller.CheckUserPermissionsForContact(database, contactID, userLogin)
		if err != nil {
			render.Render(writer, request, err) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), contactKey, contactData)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// @Summary Updates an existing notification contact to the values passed in the request body
// @ID update-contact
// @Accept json
// @Produce json
// @Param contactId path string true "ID of the contact to update"
// @Param contact body dto.Contact true "Updated contact data"
// @Success 200 {object} dto.Contact "Updated contact"
// @Failure 400 {object} api.ErrorInvalidRequestExample "Bad request from client"
// @Failure 403 {object} api.ErrorForbiddenExample "Forbidden"
// @Failure 404 {object} api.ErrorNotFoundExample "Resource not found"
// @Failure 422 {object} api.ErrorRenderExample "Render error"
// @Failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @Router /contact/{contactId} [put]
// @Tags contact
func updateContact(writer http.ResponseWriter, request *http.Request) {
	contactDTO := dto.Contact{}
	if err := render.Bind(request, &contactDTO); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	contactData := request.Context().Value(contactKey).(moira.ContactData)

	contactDTO, err := controller.UpdateContact(database, contactDTO, contactData)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, &contactDTO); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

// @Summary Deletes notification contact for the current user and remove the contact ID from all subscriptions
// @ID remove-contact
// @Accept json
// @Produce json
// @Param contactId path string true "ID of the contact to remove" Example:"bcba82f5-48cf-44c0-b7d6-e1d32c64a88c"
// @Success 200 "Contact has been deleted"
// @Failure 400 {object} api.ErrorInvalidRequestExample "Bad request from client"
// @Failure 403 {object} api.ErrorForbiddenExample "Forbidden"
// @Failure 404 {object} api.ErrorNotFoundExample "Resource not found"
// @Failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @Router /contact/{contactId} [delete]
// @Tags contact
func removeContact(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)
	err := controller.RemoveContact(database, contactData.ID, contactData.User, "")
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}

// @Summary Push a test notification to verify that the contact is properly set up
// @ID send-test-notification
// @Accept json
// @Produce json
// @Param contactId path string true "The ID of the target contact" Example:"bcba82f5-48cf-44c0-b7d6-e1d32c64a88c"
// @Success 200 "Test successfull"
// @Failure 403 {object} api.ErrorForbiddenExample "Forbidden"
// @Failure 404 {object} api.ErrorNotFoundExample "Resource not found"
// @Failure 500 {object} api.ErrorInternalServerExample "Internal server error"
// @Router /contact/{contactId}/test [post]
// @Tags contact
func sendTestContactNotification(writer http.ResponseWriter, request *http.Request) {
	contactID := middleware.GetContactID(request)
	err := controller.SendTestContactNotification(database, contactID)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}
