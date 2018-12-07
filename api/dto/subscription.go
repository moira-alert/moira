// nolint
package dto

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
)

// SubscriptionHasAnotherUserContact used when user try to save subscription with another users contacts
type SubscriptionHasAnotherUserContact struct {
	contactNames []string
}

// Error is implementation of golang error interface for SubscriptionHasAnotherUserContact struct
func (err SubscriptionHasAnotherUserContact) Error() string {
	if len(err.contactNames) == 0 {
		return fmt.Sprintf("user has not one of subscription contacts")
	}
	if len(err.contactNames) == 1 {
		return fmt.Sprintf("user has not contact '%s'", err.contactNames[0])
	}
	errBuffer := bytes.NewBuffer([]byte("user has not contacts: "))
	for idx, contactName := range err.contactNames {
		errBuffer.WriteString(fmt.Sprintf("'%s'", contactName))
		if idx != len(err.contactNames) {
			errBuffer.WriteString(", ")
		}
	}
	return errBuffer.String()
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
			return SubscriptionHasAnotherUserContact{}
		}
		anotherUserNames := make([]string, len(anotherUserContactIds))
		for _, contact := range contacts {
			anotherUserNames = append(anotherUserNames, contact.Value)
		}
		return SubscriptionHasAnotherUserContact{contactNames: anotherUserNames}
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
