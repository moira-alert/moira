package controller

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-graphite/carbonapi/date"
	"github.com/gofrs/uuid"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
)

// ErrNotAllowedContactType means that this type of contact is not allowed to be created.
var ErrNotAllowedContactType = errors.New("cannot create contact with not allowed contact type")

// GetAllContacts gets all moira contacts.
func GetAllContacts(database moira.Database) (*dto.ContactList, *api.ErrorResponse) {
	contacts, err := database.GetAllContacts()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	contactsList := dto.ContactList{
		List: make([]dto.TeamContact, 0, len(contacts)),
	}

	for _, contact := range contacts {
		contactsList.List = append(contactsList.List, dto.MakeTeamContact(contact))
	}

	return &contactsList, nil
}

// GetContactById gets notification contact by its id string.
func GetContactById(database moira.Database, contactID string) (*dto.Contact, *api.ErrorResponse) {
	contact, err := database.GetContact(contactID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	contactToReturn := &dto.Contact{
		ID:           contact.ID,
		Name:         contact.Name,
		User:         contact.User,
		TeamID:       contact.Team,
		Type:         contact.Type,
		Value:        contact.Value,
		ExtraMessage: contact.ExtraMessage,
	}

	return contactToReturn, nil
}

// CreateContact creates new notification contact for current user.
func CreateContact(
	dataBase moira.Database,
	auth *api.Authorization,
	contactsTemplate []api.WebContact,
	contact *dto.Contact,
	userLogin,
	teamID string,
) *api.ErrorResponse {
	if !isAllowedToUseContactType(auth, userLogin, contact.Type) {
		return api.ErrorInvalidRequest(ErrNotAllowedContactType)
	}

	// Only admins are allowed to create contacts for other users
	if !auth.IsAdmin(userLogin) || contact.User == "" {
		contact.User = userLogin
	}

	contactData := moira.ContactData{
		ID:           contact.ID,
		Name:         contact.Name,
		User:         contact.User,
		Team:         teamID,
		Type:         contact.Type,
		Value:        contact.Value,
		ExtraMessage: contact.ExtraMessage,
	}

	if contactData.ID == "" {
		uuid4, err := uuid.NewV4()
		if err != nil {
			return api.ErrorInternalServer(err)
		}

		contactData.ID = uuid4.String()
	} else {
		exists, err := isContactExists(dataBase, contactData.ID)
		if err != nil {
			return api.ErrorInternalServer(err)
		}

		if exists {
			return api.ErrorInvalidRequest(fmt.Errorf("contact with this ID already exists"))
		}
	}

	if err := validateContact(contactsTemplate, contactData); err != nil {
		return api.ErrorInvalidRequest(err)
	}

	if err := dataBase.SaveContact(&contactData); err != nil {
		return api.ErrorInternalServer(err)
	}

	contact.User = contactData.User
	contact.ID = contactData.ID
	contact.TeamID = contactData.Team

	return nil
}

// UpdateContact updates notification contact for current user.
func UpdateContact(
	dataBase moira.Database,
	auth *api.Authorization,
	contactsTemplate []api.WebContact,
	contactDTO dto.Contact,
	contactData moira.ContactData,
) (dto.Contact, *api.ErrorResponse) {
	if !isAllowedToUseContactType(auth, contactDTO.User, contactDTO.Type) {
		return contactDTO, api.ErrorInvalidRequest(ErrNotAllowedContactType)
	}

	contactData.Type = contactDTO.Type
	contactData.Value = contactDTO.Value
	contactData.Name = contactDTO.Name
	contactData.ExtraMessage = contactDTO.ExtraMessage

	if contactDTO.User != "" || contactDTO.TeamID != "" {
		contactData.User = contactDTO.User
		contactData.Team = contactDTO.TeamID
	}

	if err := validateContact(contactsTemplate, contactData); err != nil {
		return contactDTO, api.ErrorInvalidRequest(err)
	}

	if err := dataBase.SaveContact(&contactData); err != nil {
		return contactDTO, api.ErrorInternalServer(err)
	}

	contactDTO.User = contactData.User
	contactDTO.TeamID = contactData.Team
	contactDTO.ID = contactData.ID

	return contactDTO, nil
}

// RemoveContact deletes notification contact for current user and remove contactID from all subscriptions.
func RemoveContact(database moira.Database, contactID string, userLogin string, teamID string) *api.ErrorResponse { //nolint:gocyclo
	subscriptionIDs := make([]string, 0)

	if userLogin != "" {
		userSubscriptionIDs, err := database.GetUserSubscriptionIDs(userLogin)
		if err != nil {
			return api.ErrorInternalServer(err)
		}

		subscriptionIDs = append(subscriptionIDs, userSubscriptionIDs...)
	}

	if teamID != "" {
		teamSubscriptionIDs, err := database.GetTeamSubscriptionIDs(teamID)
		if err != nil {
			return api.ErrorInternalServer(err)
		}

		subscriptionIDs = append(subscriptionIDs, teamSubscriptionIDs...)
	}

	subscriptions, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return api.ErrorInternalServer(err)
	}

	subscriptionsWithDeletingContact := make([]*moira.SubscriptionData, 0)

	for _, subscription := range subscriptions {
		if subscription == nil {
			continue
		}

		for i, contact := range subscription.Contacts {
			if contact == contactID {
				subscription.Contacts = append(subscription.Contacts[:i], subscription.Contacts[i+1:]...)
				subscriptionsWithDeletingContact = append(subscriptionsWithDeletingContact, subscription)

				break
			}
		}
	}

	if len(subscriptionsWithDeletingContact) > 0 {
		errBuffer := bytes.NewBuffer([]byte("this contact is being used in following subscriptions: "))
		for subInd, subscription := range subscriptionsWithDeletingContact {
			errBuffer.WriteString(subscription.ID)
			errBuffer.WriteString(" (tags: ")

			for tagInd := range subscription.Tags {
				errBuffer.WriteString(subscription.Tags[tagInd])

				if tagInd != len(subscription.Tags)-1 {
					errBuffer.WriteString(", ")
				}
			}

			errBuffer.WriteString(")")

			if subInd != len(subscriptionsWithDeletingContact)-1 {
				errBuffer.WriteString(", ")
			}
		}

		return api.ErrorInvalidRequest(errors.New(errBuffer.String()))
	}

	if err := database.RemoveContact(contactID); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// SendTestContactNotification push test notification to verify the correct contact settings.
func SendTestContactNotification(dataBase moira.Database, contactID string) *api.ErrorResponse {
	eventData := &moira.NotificationEvent{
		ContactID: contactID,
		Metric:    "Test.metric.value",
		Values:    map[string]float64{"t1": 1},
		OldState:  moira.StateTEST,
		State:     moira.StateTEST,
		Timestamp: date.DateParamToEpoch("now", "", time.Now().Add(-24*time.Hour).Unix(), time.UTC),
	}
	if err := dataBase.PushNotificationEvent(eventData, false); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// CheckUserPermissionsForContact checks contact for existence and permissions for given user.
func CheckUserPermissionsForContact(
	dataBase moira.Database,
	contactID string,
	userLogin string,
	auth *api.Authorization,
) (moira.ContactData, *api.ErrorResponse) {
	contactData, err := dataBase.GetContact(contactID)
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			return moira.ContactData{}, api.ErrorNotFound(fmt.Sprintf("contact with ID '%s' does not exists", contactID))
		}

		return moira.ContactData{}, api.ErrorInternalServer(err)
	}

	if auth.IsAdmin(userLogin) {
		return contactData, nil
	}

	if contactData.Team != "" {
		teamContainsUser, err := dataBase.IsTeamContainUser(contactData.Team, userLogin)
		if err != nil {
			return moira.ContactData{}, api.ErrorInternalServer(err)
		}

		if teamContainsUser {
			return contactData, nil
		}
	}

	if contactData.User == userLogin {
		return contactData, nil
	}

	return moira.ContactData{}, api.ErrorForbidden("you are not permitted")
}

func isContactExists(dataBase moira.Database, contactID string) (bool, error) {
	_, err := dataBase.GetContact(contactID)
	if errors.Is(err, database.ErrNil) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func isAllowedToUseContactType(auth *api.Authorization, userLogin string, contactType string) bool {
	isAuthEnabled := auth.IsEnabled()
	isAdmin := auth.IsAdmin(userLogin)
	_, isAllowedContactType := auth.AllowedContactTypes[contactType]

	return isAllowedContactType || isAdmin || !isAuthEnabled
}

func validateContact(contactsTemplate []api.WebContact, contact moira.ContactData) error {
	var validationPattern string

	for _, contactTemplate := range contactsTemplate {
		if contactTemplate.ContactType == contact.Type {
			validationPattern = contactTemplate.ValidationRegex
			break
		}
	}

	if matched, err := regexp.MatchString(validationPattern, contact.Value); !matched || err != nil {
		return fmt.Errorf("contact value doesn't match regex: '%s'", validationPattern)
	}

	return nil
}

// GetContactNoisiness get contacts with amount of notification events (within time range [from, to])
// and sorts by events_count according to sortOrder.
func GetContactNoisiness(
	database moira.Database,
	page, size int64,
	from, to string,
	sortOrder api.SortOrder,
) (*dto.ContactNoisinessList, *api.ErrorResponse) {
	contacts, err := database.GetAllContacts()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	idsWithEventsCount, err := database.CountEventsInNotificationHistory(getOnlyIDs(contacts), from, to)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	noisinessSlice := makeContactNoisinessSlice(contacts, idsWithEventsCount)

	sortContactNoisinessByEventsCount(noisinessSlice, sortOrder)
	total := int64(len(noisinessSlice))

	return &dto.ContactNoisinessList{
		Page:  page,
		Size:  size,
		Total: total,
		List:  applyPagination[*dto.ContactNoisiness](page, size, total, noisinessSlice),
	}, nil
}

func getOnlyIDs(contactsData []*moira.ContactData) []string {
	ids := make([]string, 0, len(contactsData))

	for _, data := range contactsData {
		ids = append(ids, data.ID)
	}

	return ids
}

func makeContactNoisinessSlice(contacts []*moira.ContactData, idsWithEventsCount []*moira.ContactIDWithNotificationCount) []*dto.ContactNoisiness {
	noisiness := make([]*dto.ContactNoisiness, 0, len(contacts))

	for i, contact := range contacts {
		noisiness = append(noisiness,
			&dto.ContactNoisiness{
				Contact:     dto.NewContact(*contact),
				EventsCount: idsWithEventsCount[i].Count,
			})
	}

	return noisiness
}

func sortContactNoisinessByEventsCount(noisiness []*dto.ContactNoisiness, sortOrder api.SortOrder) {
	if sortOrder == api.AscSortOrder || sortOrder == api.DescSortOrder {
		slices.SortFunc(noisiness, func(first, second *dto.ContactNoisiness) int {
			cmpRes := 0
			if first.EventsCount > second.EventsCount {
				cmpRes = 1
			} else if second.EventsCount > first.EventsCount {
				cmpRes = -1
			}

			if cmpRes == 0 {
				return strings.Compare(first.ID, second.ID)
			}

			if sortOrder == api.DescSortOrder {
				cmpRes *= -1
			}

			return cmpRes
		})
	}
}
