// nolint
package dto

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
)

// UserNotPermittedToUseContactsError used when user try to save subscription with another users contacts
type UserNotPermittedToUseContactsError struct {
	contactIds   []string
	contactNames []string
}

// Error is implementation of golang error interface for UserNotPermittedToUseContactsError struct
func (err UserNotPermittedToUseContactsError) Error() string {
	if len(err.contactNames) == 1 {
		return fmt.Sprintf("user not permitted to use contact '%s'", err.contactNames[0])
	}
	if len(err.contactNames) > 1 {
		return fmt.Sprintf("user not permitted to use following contacts: %s", strings.Join(err.contactNames, ", "))
	}
	if len(err.contactIds) == 1 {
		return fmt.Sprintf("failed to identify the ownership of the contact id '%s'", err.contactIds[0])
	}
	if len(err.contactIds) > 1 {
		return fmt.Sprintf("failed to identify the ownership of the following contact ids: '%s'", strings.Join(err.contactIds, ", "))
	}
	return "failed to identify the ownership of requested contacts"
}

type SubscriptionList struct {
	List []moira.SubscriptionData `json:"list"`
}

func (*SubscriptionList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Subscription moira.SubscriptionData

func (*Subscription) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (subscription *Subscription) Bind(request *http.Request) error {
	subscription.Tags = normalizeTags(subscription.Tags)
	if len(subscription.Tags) == 0 {
		return fmt.Errorf("subscription must have tags")
	}
	if len(subscription.Contacts) == 0 {
		return fmt.Errorf("subscription must have contacts")
	}
	return subscription.checkContacts(request)
}

func (subscription *Subscription) checkContacts(request *http.Request) error {
	database := middleware.GetDatabase(request)
	userLogin := middleware.GetLogin(request)
	contactIDs, err := database.GetUserContactIDs(userLogin)
	if err != nil {
		return err
	}

	userContactIdsHash := make(map[string]interface{})
	for _, contactId := range contactIDs {
		userContactIdsHash[contactId] = true
	}

	anotherUserContactIds := make([]string, 0)
	for _, subContactId := range subscription.Contacts {
		if _, ok := userContactIdsHash[subContactId]; !ok {
			anotherUserContactIds = append(anotherUserContactIds, subContactId)
		}
	}
	if len(anotherUserContactIds) > 0 {
		contacts, err := database.GetContacts(anotherUserContactIds)
		if err != nil {
			return UserNotPermittedToUseContactsError{contactIds: anotherUserContactIds}
		}
		anotherUserNames := make([]string, 0)
		anotherContactIds := make([]string, 0)
		for i, contact := range contacts {
			if contact == nil {
				anotherContactIds = append(anotherContactIds, anotherUserContactIds[i])
			} else {
				anotherUserNames = append(anotherUserNames, contact.Value)
			}
		}
		return UserNotPermittedToUseContactsError{
			contactNames: anotherUserNames,
			contactIds:   anotherUserContactIds,
		}
	}
	return nil
}

func normalizeTags(tags []string) []string {
	var normalized = make([]string, 0)
	for _, subTag := range tags {
		if subTag != "" {
			normalized = append(normalized, subTag)
		}
	}
	return normalized
}
