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
		router.Get("/", getContactById)
		router.Put("/", updateContact)
		router.Delete("/", removeContact)
		router.Post("/test", sendTestContactNotification)
	})
}

// nolint: gofmt,goimports
//
//	@summary	Gets all Moira contacts
//	@id			get-all-contacts
//	@tags		contact
//	@produce	json
//	@success	200	{object}	dto.ContactList					"Contacts fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/contact [get]
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

// nolint: gofmt,goimports
//
//	@summary	Get contact by ID
//	@id			get-contact-by-id
//	@tags		contact
//	@produce	json
//	@param		contactID	path		string							true	"Contact ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			{object}	dto.Contact						"Successfully received contact"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/contact/{contactID} [get]
func getContactById(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)

	contact, apiErr := controller.GetContactById(database, contactData.ID)

	if apiErr != nil {
		render.Render(writer, request, apiErr) //nolint
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Creates a new contact notification for the current user
//	@id			create-new-contact
//	@tags		contact
//	@accept		json
//	@produce	json
//	@param		contact	body		dto.Contact						true	"Contact data"
//	@success	200		{object}	dto.Contact						"Contact created successfully"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/contact [put]
func createNewContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	userLogin := middleware.GetLogin(request)

	if err := controller.CreateContact(database, contact, userLogin, contact.TeamID); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// contactFilter is middleware for check contact existence and user permissions.
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

// nolint: gofmt,goimports
//
//	@summary	Updates an existing notification contact to the values passed in the request body
//	@id			update-contact
//	@accept		json
//	@produce	json
//	@param		contactID	path		string							true	"ID of the contact to update"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		contact		body		dto.Contact						true	"Updated contact data"
//	@success	200			{object}	dto.Contact						"Updated contact"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/contact/{contactID} [put]
//	@tags		contact
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

// nolint: gofmt,goimports
//
//	@summary	Deletes notification contact for the current user and remove the contact ID from all subscriptions
//	@id			remove-contact
//	@accept		json
//	@produce	json
//	@tags		contact
//	@param		contactID	path	string	true	"ID of the contact to remove"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			"Contact has been deleted"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/contact/{contactID} [delete]
func removeContact(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)
	err := controller.RemoveContact(database, contactData.ID, contactData.User, contactData.Team)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}

// nolint: gofmt,goimports
//
//	@summary	Push a test notification to verify that the contact is properly set up
//	@id			send-test-contact-notification
//	@accept		json
//	@produce	json
//	@param		contactID	path	string	true	"The ID of the target contact"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			"Test successful"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/contact/{contactID}/test [post]
//	@tags		contact
func sendTestContactNotification(writer http.ResponseWriter, request *http.Request) {
	contactID := middleware.GetContactID(request)
	err := controller.SendTestContactNotification(database, contactID)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}
