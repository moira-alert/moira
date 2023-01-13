// nolint
package dto

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
)

// ErrSubscriptionContainsTeamAndUser used when user try to save subscription team and user attributes specified
type ErrSubscriptionContainsTeamAndUser struct {
}

// Error is an error interface implementation method
func (ErrSubscriptionContainsTeamAndUser) Error() string {
	return "cannot create subscription that contains contact and team attributes"
}

// ErrProvidedContactsForbidden used when user try to save subscription with another users contacts
type ErrProvidedContactsForbidden struct {
	contactIds   []string
	contactNames []string
}

// Error is implementation of golang error interface for ErrProvidedContactsForbidden struct
func (err ErrProvidedContactsForbidden) Error() string {
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
	if len(subscription.Tags) == 0 && !subscription.AnyTags {
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
	teamID := middleware.GetTeamID(request)
	if teamID == "" && subscription.TeamID != "" {
		teamID = subscription.TeamID
	}
	if subscription.User != "" && teamID != "" {
		return ErrSubscriptionContainsTeamAndUser{}
	}
	var contactIDs []string
	var err error
	if teamID != "" {
		contactIDs, err = database.GetTeamContactIDs(teamID)
	} else {
		contactIDs, err = database.GetUserContactIDs(userLogin)
	}
	if err != nil {
		return err
	}

	contactIDsHash := make(map[string]interface{})
	for _, contactId := range contactIDs {
		contactIDsHash[contactId] = true
	}

	subscriptionContactIDs := make([]string, 0)
	for _, subContactId := range subscription.Contacts {
		if _, ok := contactIDsHash[subContactId]; !ok {
			subscriptionContactIDs = append(subscriptionContactIDs, subContactId)
		}
	}

	if len(subscriptionContactIDs) > 0 {
		contacts, err := database.GetContacts(subscriptionContactIDs)
		if err != nil {
			return ErrProvidedContactsForbidden{contactIds: subscriptionContactIDs}
		}
		anotherUserContactValues := make([]string, 0)
		anotherUserContactIDs := make([]string, 0)
		for i, contact := range contacts {
			if contact == nil {
				anotherUserContactIDs = append(anotherUserContactIDs, subscriptionContactIDs[i])
			} else {
				anotherUserContactValues = append(anotherUserContactValues, contact.Value)
			}
		}
		return ErrProvidedContactsForbidden{
			contactNames: anotherUserContactValues,
			contactIds:   subscriptionContactIDs,
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
