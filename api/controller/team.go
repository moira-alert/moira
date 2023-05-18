package controller

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-redis/redis/v8"
)

const teamIDCreateRetries = 3

// CreateTeam is a controller function that creates a new team in Moira
func CreateTeam(dataBase moira.Database, team dto.TeamModel, userID string) (dto.SaveTeamResponse, *api.ErrorResponse) {
	var teamID string
	if team.ID != "" { // if teamID is specified in request data then check that team with this id is not exist
		teamID = team.ID
		_, err := dataBase.GetTeam(teamID)
		if err == nil {
			return dto.SaveTeamResponse{}, api.ErrorInvalidRequest(fmt.Errorf("team with ID you specified already exists %s", teamID))
		}
		if err != nil && err != database.ErrNil {
			return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot check id for team: %w", err))
		}
	} else { // on the other hand try to create an UUID for teamID
		createdSuccessfully := false
		for i := 0; i < teamIDCreateRetries; i++ { // trying three times to create an UUID and check if it exists
			generatedUUID, err := uuid.NewV4()
			if err != nil {
				return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot generate id for team: %w", err))
			}
			teamID = generatedUUID.String()
			_, err = dataBase.GetTeam(teamID)
			if err == database.ErrNil {
				createdSuccessfully = true
				break
			}
			if err != nil {
				return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot check id for team: %w", err))
			}
		}
		if !createdSuccessfully {
			return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot generate unique id for team"))
		}
	}
	err := dataBase.SaveTeam(teamID, team.ToMoiraTeam())
	if err != nil {
		return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot save team: %w", err))
	}

	teamsMap, apiErr := addTeamsForNewUsers(dataBase, teamID, map[string]bool{userID: true}, map[string][]string{})
	if err != nil {
		return dto.SaveTeamResponse{}, apiErr
	}

	err = dataBase.SaveTeamsAndUsers(teamID, []string{userID}, teamsMap)
	if err != nil {
		return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot save team users: %w", err))
	}

	return dto.SaveTeamResponse{ID: teamID}, nil
}

// GetTeam is a controller function that returns a team by it's ID
func GetTeam(dataBase moira.Database, teamID string) (dto.TeamModel, *api.ErrorResponse) {
	team, err := dataBase.GetTeam(teamID)

	if err != nil {
		if err == database.ErrNil {
			return dto.TeamModel{}, api.ErrorNotFound(fmt.Sprintf("cannot find team: %s", teamID))
		}
		return dto.TeamModel{}, api.ErrorInternalServer(fmt.Errorf("cannot get team from database: %w", err))
	}

	teamModel := dto.NewTeamModel(team)
	return teamModel, nil
}

// GetUserTeams is a controller function that returns a teams in which user is a member bu user ID
func GetUserTeams(dataBase moira.Database, userID string) (dto.UserTeams, *api.ErrorResponse) {
	teams, err := dataBase.GetUserTeams(userID)

	result := []dto.TeamModel{}
	if err != nil && err != database.ErrNil {
		return dto.UserTeams{}, api.ErrorInternalServer(fmt.Errorf("cannot get user teams from database: %w", err))
	}

	for _, teamID := range teams {
		team, err := dataBase.GetTeam(teamID)
		if err != nil {
			return dto.UserTeams{}, api.ErrorInternalServer(fmt.Errorf("cannot retrieve team from database: %w", err))
		}
		teamModel := dto.NewTeamModel(team)
		result = append(result, teamModel)
	}

	return dto.UserTeams{Teams: result}, nil
}

// GetTeamUsers is a controller function that returns a users of team by team ID
func GetTeamUsers(dataBase moira.Database, teamID string) (dto.TeamMembers, *api.ErrorResponse) {
	users, err := dataBase.GetTeamUsers(teamID)

	if err != nil {
		if err == database.ErrNil {
			return dto.TeamMembers{}, api.ErrorNotFound(fmt.Sprintf("cannot find team users: %s", teamID))
		}
		return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", err))
	}

	result := dto.TeamMembers{
		Usernames: users,
	}
	return result, nil
}

func fillCurrentUsersTeamsMap(dataBase moira.Database, existingUsers []string) (map[string][]string, *api.ErrorResponse) {
	result := map[string][]string{}
	for _, userID := range existingUsers {
		fetchedUserTeams, err := dataBase.GetUserTeams(userID)
		if err != nil {
			return nil, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", err))
		}
		result[userID] = fetchedUserTeams
	}
	return result, nil
}

func removeDeletedUsers(teamID string, existingUsers []string, newUsers map[string]bool, teamsMap map[string][]string) (map[string][]string, *api.ErrorResponse) {
	for _, userID := range existingUsers {
		if newUsers[userID] {
			continue
		}
		userRemovedTeams, err := removeUserTeam(teamsMap[userID], teamID)
		if err != nil {
			return nil, api.ErrorInternalServer(fmt.Errorf("cannot remove team from user: %w", err))
		}
		teamsMap[userID] = userRemovedTeams
	}
	return teamsMap, nil
}

func addTeamsForNewUsers(dataBase moira.Database, teamID string, newUsers map[string]bool, teamsMap map[string][]string) (map[string][]string, *api.ErrorResponse) {
	for userID := range newUsers {
		// Skip users that already were in this team
		if _, ok := teamsMap[userID]; ok {
			continue
		}
		fetchedUserTeams, err := dataBase.GetUserTeams(userID) //nolint:govet
		if err != nil && err != database.ErrNil {
			return nil, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", err))
		}
		fetchedUserTeams = append(fetchedUserTeams, teamID)
		teamsMap[userID] = fetchedUserTeams
	}
	return teamsMap, nil
}

// SetTeamUsers is a controller function that sets all users for team
func SetTeamUsers(dataBase moira.Database, teamID string, allUsers []string) (dto.TeamMembers, *api.ErrorResponse) {
	existingUsers, err := dataBase.GetTeamUsers(teamID)
	if err != nil {
		if err == database.ErrNil {
			return dto.TeamMembers{}, api.ErrorNotFound(fmt.Sprintf("cannot find team users: %s", teamID))
		}
		return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", err))
	}
	/*
		This map will contain final list of all users and all teams of each user that were affected by this update.
		After we will have a map like
		{
			"existingUser1": {"currentTeam", "team1", "team2"},
			"userThatWillBeDeleted": {"currentTeam", "team2"}
		}
	*/
	teamsMap, apiError := fillCurrentUsersTeamsMap(dataBase, existingUsers)
	if apiError != nil {
		return dto.TeamMembers{}, apiError
	}

	allUsersMap := map[string]bool{}

	// Collect a set of all new users
	for _, userID := range allUsers {
		allUsersMap[userID] = true
	}

	/* Here we will find a users that do not exist in new users list
	and remove their teams from
	after that our teams map will be like this:
	{
		"existingUser1": {"currentTeam", "team1", "team2"},
		"userThatWillBeDeleted": {"team2"}
	}
	*/
	teamsMap, apiError = removeDeletedUsers(teamID, existingUsers, allUsersMap, teamsMap)
	if apiError != nil {
		return dto.TeamMembers{}, apiError
	}

	/*
		For all new users we need to retrieve their actual teams and add current team after thi teams map will look like this:
			{
			"existingUser1": {"currentTeam", "team1", "team2"},
			"userThatWillBeDeleted": {"team2"},
			"newUser": {"existingTeam", "currentTeam"}
		}
	*/
	teamsMap, apiError = addTeamsForNewUsers(dataBase, teamID, allUsersMap, teamsMap)
	if apiError != nil {
		return dto.TeamMembers{}, apiError
	}

	err = dataBase.SaveTeamsAndUsers(teamID, allUsers, teamsMap)
	if err != nil {
		api.ErrorInternalServer(fmt.Errorf("cannot save users for team: %s %w", teamID, err))
	}

	result := dto.TeamMembers{
		Usernames: allUsers,
	}
	return result, nil
}

func addUserTeam(teamID string, teams []string) ([]string, error) {
	for _, currentTeamID := range teams {
		if teamID == currentTeamID {
			return []string{}, fmt.Errorf("team already exist in user teams: %s", teamID)
		}
	}
	teams = append(teams, teamID)
	return teams, nil
}

// AddTeamUsers is a controller function that adds a users to certain team
func AddTeamUsers(dataBase moira.Database, teamID string, newUsers []string) (dto.TeamMembers, *api.ErrorResponse) {
	existingUsers, err := dataBase.GetTeamUsers(teamID)
	if err != nil {
		if err == database.ErrNil {
			return dto.TeamMembers{}, api.ErrorNotFound(fmt.Sprintf("cannot find team users: %s", teamID))
		}
		return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", err))
	}

	teamsMap := map[string][]string{}
	finalUsers := []string{}

	for _, userID := range existingUsers {
		userTeams, err := dataBase.GetUserTeams(userID) //nolint:govet
		if err != nil {
			if err == database.ErrNil {
				return dto.TeamMembers{}, api.ErrorNotFound(fmt.Sprintf("cannot find user teams: %s", userID))
			}
			return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot get user teams from database: %w", err))
		}
		teamsMap[userID] = userTeams
		finalUsers = append(finalUsers, userID)
	}

	for _, userID := range newUsers {
		if _, ok := teamsMap[userID]; ok {
			return dto.TeamMembers{}, api.ErrorInvalidRequest(fmt.Errorf("one ore many users you specified are already exist in this team: %s", userID))
		}

		userTeams, err := dataBase.GetUserTeams(userID) //nolint:govet
		if err != nil && err != redis.Nil {
			return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot get user teams from database: %w", err))
		}

		userTeams, err = addUserTeam(teamID, userTeams)
		if err != nil {
			return dto.TeamMembers{}, api.ErrorInvalidRequest(fmt.Errorf("cannot save new team for user: %s, %w", userID, err))
		}

		teamsMap[userID] = userTeams
		finalUsers = append(finalUsers, userID)
	}

	err = dataBase.SaveTeamsAndUsers(teamID, finalUsers, teamsMap)
	if err != nil {
		api.ErrorInternalServer(fmt.Errorf("cannot save users for team: %s %w", teamID, err))
	}

	result := dto.TeamMembers{
		Usernames: finalUsers,
	}
	return result, nil
}

// UpdateTeam is a controller function that updates an existing team in Moira
func UpdateTeam(dataBase moira.Database, teamID string, team dto.TeamModel) (dto.SaveTeamResponse, *api.ErrorResponse) {
	err := dataBase.SaveTeam(teamID, team.ToMoiraTeam())
	if err != nil {
		return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot save team: %w", err))
	}
	return dto.SaveTeamResponse{ID: teamID}, nil
}

// DeleteTeam is a controller function that removes an existing team in Moira
func DeleteTeam(dataBase moira.Database, teamID, userLogin string) (dto.SaveTeamResponse, *api.ErrorResponse) {
	teamUsers, err := dataBase.GetTeamUsers(teamID)
	if err != nil {
		return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot get team users: %w", err))
	}
	if len(teamUsers) > 1 {
		return dto.SaveTeamResponse{}, api.ErrorInvalidRequest(fmt.Errorf("cannot delete team: team have users: %s", strings.Join(teamUsers, ", ")))
	}
	teamContacts, err := dataBase.GetTeamContactIDs(teamID)
	if err != nil {
		return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot get team contacts: %w", err))
	}
	if len(teamContacts) > 0 {
		return dto.SaveTeamResponse{}, api.ErrorInvalidRequest(fmt.Errorf("cannot delete team: team have contacts: %s", strings.Join(teamContacts, ", ")))
	}
	teamSubscriptions, err := dataBase.GetTeamSubscriptionIDs(teamID)
	if err != nil {
		return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot get team subscriptions: %w", err))
	}
	if len(teamSubscriptions) > 0 {
		return dto.SaveTeamResponse{}, api.ErrorInvalidRequest(fmt.Errorf("cannot delete team: team have subscriptions: %s", strings.Join(teamSubscriptions, ", ")))
	}
	err = dataBase.DeleteTeam(teamID, userLogin)
	if err != nil {
		return dto.SaveTeamResponse{}, api.ErrorInternalServer(fmt.Errorf("cannot delete team: %w", err))
	}
	return dto.SaveTeamResponse{ID: teamID}, nil
}

// DeleteTeamUser is a controller function that removes a user from certain team
func DeleteTeamUser(dataBase moira.Database, teamID string, removeUserID string) (dto.TeamMembers, *api.ErrorResponse) {
	existingUsers, err := dataBase.GetTeamUsers(teamID)
	if err != nil {
		if err == database.ErrNil {
			return dto.TeamMembers{}, api.ErrorNotFound(fmt.Sprintf("cannot find team users: %s", teamID))
		}
		return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", err))
	}

	if len(existingUsers) <= 1 {
		return dto.TeamMembers{}, api.ErrorInvalidRequest(fmt.Errorf("cannot remove last member of team"))
	}

	userFound := false
	for _, userID := range existingUsers {
		if userID == removeUserID {
			userFound = true
		}
	}
	if !userFound {
		return dto.TeamMembers{}, api.ErrorNotFound(fmt.Sprintf("user that you specified not found in this team: %s", removeUserID))
	}

	teamsMap := map[string][]string{}
	finalUsers := []string{}

	for _, userID := range existingUsers {
		userTeams, err := dataBase.GetUserTeams(userID) //nolint:govet
		if err != nil {
			if err == database.ErrNil {
				return dto.TeamMembers{}, api.ErrorNotFound(fmt.Sprintf("cannot find user teams: %s", userID))
			}
			return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot get user teams from database: %w", err))
		}
		if userID == removeUserID {
			userTeams, err = removeUserTeam(userTeams, teamID)
			if err != nil {
				return dto.TeamMembers{}, api.ErrorInternalServer(fmt.Errorf("cannot remove team from user: %w", err))
			}
		} else {
			finalUsers = append(finalUsers, userID)
		}
		teamsMap[userID] = userTeams
	}

	err = dataBase.SaveTeamsAndUsers(teamID, finalUsers, teamsMap)
	if err != nil {
		api.ErrorInternalServer(fmt.Errorf("cannot save users for team: %s %w", teamID, err))
	}

	result := dto.TeamMembers{
		Usernames: finalUsers,
	}
	return result, nil
}

func removeUserTeam(teams []string, teamID string) ([]string, error) {
	for i, currentTeamID := range teams {
		if teamID == currentTeamID {
			teams[i] = teams[len(teams)-1]   // Copy last element to index i.
			teams[len(teams)-1] = ""         // Erase last element (write zero value).
			return teams[:len(teams)-1], nil // Truncate slice.
		}
	}
	return []string{}, fmt.Errorf("cannot find team in user teams: %s", teamID)
}

func CheckUserPermissionsForTeam(dataBase moira.Database, teamID, userID string) *api.ErrorResponse {
	_, err := dataBase.GetTeam(teamID)
	if err != nil {
		if err == database.ErrNil {
			return api.ErrorNotFound(fmt.Sprintf("team with ID '%s' does not exists", teamID))
		}
		return api.ErrorInternalServer(err)
	}

	userIsTeamMember, err := dataBase.IsTeamContainUser(teamID, userID)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	if !userIsTeamMember {
		return api.ErrorForbidden("you are not permitted to manipulate with this team")
	}
	return nil
}

// GetTeamSettings gets team contacts and subscriptions
func GetTeamSettings(database moira.Database, teamID string) (dto.TeamSettings, *api.ErrorResponse) {
	teamSettings := dto.TeamSettings{
		TeamID:        teamID,
		Contacts:      make([]moira.ContactData, 0),
		Subscriptions: make([]moira.SubscriptionData, 0),
	}

	subscriptionIDs, err := database.GetTeamSubscriptionIDs(teamID)
	if err != nil {
		return dto.TeamSettings{}, api.ErrorInternalServer(err)
	}

	subscriptions, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return dto.TeamSettings{}, api.ErrorInternalServer(err)
	}
	for _, subscription := range subscriptions {
		if subscription != nil {
			teamSettings.Subscriptions = append(teamSettings.Subscriptions, *subscription)
		}
	}
	contactIDs, err := database.GetTeamContactIDs(teamID)
	if err != nil {
		return dto.TeamSettings{}, api.ErrorInternalServer(err)
	}

	contacts, err := database.GetContacts(contactIDs)
	if err != nil {
		return dto.TeamSettings{}, api.ErrorInternalServer(err)
	}
	for _, contact := range contacts {
		if contact != nil {
			teamSettings.Contacts = append(teamSettings.Contacts, *contact)
		}
	}
	return teamSettings, nil
}

// GetTeamSubsStats return teams with subscriptions statistics.
func GetTeamSubsStats(database moira.Database, logger moira.Logger) (dto.TeamSubsStats, error) {
	teams, err := database.GetAllTeams()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(teams))
	statsChan := make(chan *dto.TeamSubsStatsElement, len(teams))

	for i, team := range teams {
		wg.Add(1)
		i := i
		go func(team *moira.Team) {
			logger.Info().Msg(fmt.Sprintf("Started i=%d, team=%s", i, team.Name))

			defer wg.Done()
			se, createErr := createStatElement(database, *team)
			if createErr != nil {
				errChan <- createErr
			}
			statsChan <- se
			logger.Info().Msg(fmt.Sprintf("Finished i=%d", i))
		}(team)
	}

	wg.Wait()
	close(statsChan)
	close(errChan)

	for err = range errChan {
		logger.Error().Msg(err.Error())
	}

	stats := make(dto.TeamSubsStats, 0)
	for s := range statsChan {
		stats = append(stats, s)
	}
	return stats, nil
}

func createStatElement(database moira.Database, team moira.Team) (*dto.TeamSubsStatsElement, error) {
	se := dto.TeamSubsStatsElement{}
	se.TeamID = team.ID
	se.TeamName = team.Name

	subIDs, err := database.GetTeamSubscriptionIDs(team.ID)
	if err != nil {
		return nil, err
	}
	se.SubscriptionsCount = len(subIDs)

	subs, err := database.GetSubscriptions(subIDs)
	if err != nil {
		return nil, err
	}
	uniqueTags := mapset.NewSet[string]()
	for _, sub := range subs {
		for _, tag := range sub.Tags {
			uniqueTags.Add(tag)
		}
	}
	se.UniqueTagsCount = uniqueTags.Cardinality()

	contactIDs, err := database.GetTeamContactIDs(team.ID)
	if err != nil {
		return nil, err
	}
	se.ContactsCount = len(contactIDs)

	users, err := database.GetTeamUsers(team.ID)
	if err != nil {
		return nil, err
	}
	se.UsersCount = len(users)

	contacts, err := database.GetContacts(contactIDs)
	if err != nil {
		return nil, err
	}
	uniqueSendersCount := mapset.NewSet[string]()
	for _, contact := range contacts {
		uniqueSendersCount.Add(contact.Type)
	}
	se.UniqueSendersCount = uniqueSendersCount.Cardinality()

	return &se, nil
}

type teamIDTriggerIDs map[string]mapset.Set[string]

// GetTeamTriggersStats return teams with triggers statistics.
func GetTeamTriggersStats(database moira.Database, logger moira.Logger) (dto.TeamTriggersStats, *api.ErrorResponse) {
	tagsNames, err := database.GetTagNames()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	resultChan := make(chan teamIDTriggerIDs, len(tagsNames))
	errChan := make(chan error, len(tagsNames))
	var wg sync.WaitGroup
	var handledTagsCount uint32

	for i, tagName := range tagsNames {
		i := i
		wg.Add(1)
		go func(tagName string) {
			defer wg.Done()
			defer func() {
				atomic.AddUint32(&handledTagsCount, 1)
				logger.Info().Msg(fmt.Sprintf("handledTagsCount=%d", atomic.LoadUint32(&handledTagsCount)))
			}()

			logger.Info().Msg(fmt.Sprintf("Started i=%d, tag=%s", i, tagName))

			teamTriggers, tagError := getTeamTriggersByTag(database, tagName)
			if tagError != nil {
				errChan <- tagError
				logger.Info().Msg(fmt.Sprintf("Finished with err=%s, i=%d", tagError, i))

				return
			}
			resultChan <- teamTriggers
			logger.Info().Msg(fmt.Sprintf("Finished i=%d, tag=%s", i, tagName))
		}(tagName)
	}
	wg.Wait()
	close(errChan)
	close(resultChan)
	for err = range errChan {
		logger.Error().Msg(err.Error())
		return nil, api.ErrorInternalServer(err)
	}

	allTeams, err := database.GetAllTeams()
	if err != nil {
		logger.Error().Msg(err.Error())
		return nil, api.ErrorInternalServer(err)
	}
	// create one map from few maps
	teamIDToTriggerIDs := make(teamIDTriggerIDs, 0)
	for _, team := range allTeams {
		teamIDToTriggerIDs[team.ID] = mapset.NewSet[string]()
	}
	for stats := range resultChan {
		for teamID, triggerIDs := range stats {
			for triggerID := range triggerIDs.Iter() {
				teamIDToTriggerIDs[teamID].Add(triggerID)
			}
		}
	}

	stats := make(dto.TeamTriggersStats, 0)
	for _, team := range allTeams {
		element := &dto.TeamTriggersStatsElement{
			TeamID:        team.ID,
			TeamName:      team.Name,
			TriggersCount: teamIDToTriggerIDs[team.ID].Cardinality(),
		}
		stats = append(stats, element)
	}

	return stats, nil
}

func getTeamTriggersByTag(database moira.Database, tag string) (teamIDTriggerIDs, error) {
	subs, err := database.GetTagsSubscriptions([]string{tag})
	if err != nil {
		return nil, err
	}

	triggerIDs, err := database.GetTagTriggerIDs(tag)
	if err != nil {
		return nil, err
	}

	teamIDToTriggerIDs := make(teamIDTriggerIDs)
	for _, sub := range subs {
		if sub == nil || !sub.Enabled || sub.TeamID == "" {
			continue
		}
		subTags := mapset.NewSet(sub.Tags...)

		for _, triggerID := range triggerIDs {
			trigger, err := database.GetTrigger(triggerID)
			if err != nil {
				return nil, err
			}
			triggerTags := mapset.NewSet(trigger.Tags...)
			if isTriggerRelatesToSubscription := subTags.IsSubset(triggerTags); isTriggerRelatesToSubscription {
				if teamIDToTriggerIDs[sub.TeamID] == nil {
					teamIDToTriggerIDs[sub.TeamID] = mapset.NewSet[string]()
				}
				teamIDToTriggerIDs[sub.TeamID].Add(triggerID)
			}
		}
	}

	return teamIDToTriggerIDs, nil
}
