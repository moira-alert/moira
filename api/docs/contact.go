package docs

import (
	"github.com/moira-alert/moira/api/dto"
)

// swagger:route GET /contact contacts getAllContacts
//
// Returns all contacts.
//     Produces:
//     - application/json
//
//     Responses:
//       200: contactResponse
//       422: errorResponse
//       500: errorResponse

// swagger:response contactResponse
type getContactsResponse struct {
	// in:body
	Body dto.ContactList
}

// swagger:route PUT /contact contacts createNewContact
//
// Creates new notification contact for current user.
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       200: contactResponse
//       400: errorResponse
//       422: errorResponse
//       500: errorResponse

// swagger:parameters createNewContact
type createNewContactRequest struct {
	// in:body
	Body dto.Contact
}

// swagger:response contactResponse
type createNewContactResponse struct {
	// in:body
	Body dto.Contact
}

// swagger:route PUT /contact/{contactId} contacts updateContact
//
// Updates notification contact for current user.
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       200: contactResponse
//       400: errorResponse
//       422: errorResponse
//       500: errorResponse

// swagger:parameters updateContact
type updateContactRequest struct {
	// in:path
	Path contactIdContext
	// in:body
	Body dto.Contact
}

// swagger:response contactResponse
type updateContactResponse struct {
	// in:body
	Body dto.Contact
}

// swagger:route DELETE /contact/{contactId} contacts deleteContact
//
// Deletes notification contact for current user.
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       200:
//       400: errorResponse
//       500: errorResponse

// swagger:parameters deleteContact
type deleteContactRequest struct {
	// in:path
	Path contactIdContext
}

// swagger:response contactResponse
type deleteContactResponse struct {
	// in:body
	Body dto.Contact
}
