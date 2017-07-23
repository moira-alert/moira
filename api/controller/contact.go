package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/satori/go.uuid"
)

func GetAllContacts(database moira.Database) (*dto.ContactList, *dto.ErrorResponse) {
	contacts, err := database.GetAllContacts()
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}
	contactsList := dto.ContactList{
		List: contacts,
	}
	return &contactsList, nil
}

func CreateContact(database moira.Database, contact *dto.Contact, userLogin string) *dto.ErrorResponse {
	id := uuid.NewV4().String()
	contactData := &moira.ContactData{
		ID:    id,
		User:  userLogin,
		Type:  contact.Type,
		Value: contact.Value,
	}

	if err := database.WriteContact(contactData); err != nil {
		return dto.ErrorInternalServer(err)
	}

	contact.ID = &id
	contact.User = &userLogin
	return nil
}

func DeleteContact(database moira.Database, contactId string, userLogin string) *dto.ErrorResponse {
	subscriptionIds, err := database.GetUserSubscriptionIds(userLogin)
	if err != nil {
		return dto.ErrorInternalServer(err)
	}

	subscriptions, err := database.GetSubscriptions(subscriptionIds)
	if err != nil {
		return dto.ErrorInternalServer(err)
	}

	subscriptionsWithDeletingContact := make([]moira.SubscriptionData, 0)

	for _, subscription := range subscriptions {
		for i, contact := range subscription.Contacts {
			if contact == contactId {
				subscription.Contacts = append(subscription.Contacts[:i], subscription.Contacts[i+1:]...)
				subscriptionsWithDeletingContact = append(subscriptionsWithDeletingContact, subscription)
				break
			}
		}
	}

	if err := database.DeleteContact(contactId, userLogin); err != nil {
		return dto.ErrorInternalServer(err)
	}

	if err := database.WriteSubscriptions(subscriptionsWithDeletingContact); err != nil {
		return dto.ErrorInternalServer(err)
	}

	return nil
}
