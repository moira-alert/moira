package controller

import (
	"fmt"
	"math"
	"regexp"

	"github.com/gofrs/uuid"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
)

const pageSizeUnlimited int64 = -1

var idValidationPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

// CreateTrigger creates new trigger
func CreateTrigger(dataBase moira.Database, trigger *dto.TriggerModel, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	if trigger.ID == "" {
		uuid4, err := uuid.NewV4()
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		trigger.ID = uuid4.String()
	} else {
		if !idValidationPattern.MatchString(trigger.ID) {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger ID contains invalid characters (allowed: 0-9, a-z, A-Z, -, ~, _, .)"))
		}
		exists, err := triggerExists(dataBase, trigger.ID)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		if exists {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger with this ID already exists"))
		}
	}
	resp, err := saveTrigger(dataBase, trigger.ToMoiraTrigger(), trigger.ID, timeSeriesNames)
	if resp != nil {
		resp.Message = "trigger created"
	}
	return resp, err
}

// GetAllTriggers gets all moira triggers
func GetAllTriggers(database moira.Database) (*dto.TriggersList, *api.ErrorResponse) {
	triggerIDs, err := database.GetAllTriggerIDs()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggersList := dto.TriggersList{
		List: make([]moira.TriggerCheck, 0),
	}
	for _, triggerCheck := range triggerChecks {
		if triggerCheck != nil {
			triggersList.List = append(triggersList.List, *triggerCheck)
		}
	}
	return &triggersList, nil
}

// SearchTriggers gets trigger page and filter trigger by tags and search request terms
func SearchTriggers(database moira.Database, searcher moira.Searcher, options moira.SearchOptions, createPager bool, pagerID string) (*dto.TriggersList, *api.ErrorResponse) { //nolint
	var searchResults []*moira.SearchResult
	var total int64
	pagerShouldExist := pagerID != ""

	if pagerShouldExist && (options.SearchString != "" || len(options.Tags) > 0) {
		return nil, api.ErrorInvalidRequest(fmt.Errorf("cannot handle request with search string or tags and pager ID set"))
	}
	if pagerShouldExist {
		var err error
		searchResults, total, err = database.GetTriggersSearchResults(pagerID, options.Page, options.Size)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		if searchResults == nil {
			return nil, api.ErrorNotFound("Pager not found")
		}
	} else {
		var err error
		if createPager {
			options.Size = pageSizeUnlimited
		}
		searchResults, total, err = searcher.SearchTriggers(options)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
	}

	if createPager && !pagerShouldExist {
		uuid4, err := uuid.NewV4()
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		pagerID = uuid4.String()
		database.SaveTriggersSearchResults(pagerID, searchResults) //nolint
	}

	if createPager {
		var from, to int64 = 0, int64(len(searchResults))
		if options.Size >= 0 {
			from = int64(math.Min(float64(options.Page*options.Size), float64(len(searchResults))))
			to = int64(math.Min(float64(from+options.Size), float64(len(searchResults))))
		}
		searchResults = searchResults[from:to]
	}

	var triggerIDs []string //nolint
	for _, searchResult := range searchResults {
		triggerIDs = append(triggerIDs, searchResult.ObjectID)
	}

	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	var pagerIDPtr *string
	if pagerID != "" {
		pagerIDPtr = &pagerID
	}

	triggersList := dto.TriggersList{
		List:  make([]moira.TriggerCheck, 0),
		Total: &total,
		Page:  &options.Page,
		Size:  &options.Size,
		Pager: pagerIDPtr,
	}

	for triggerCheckInd := range triggerChecks {
		triggerCheck := triggerChecks[triggerCheckInd]
		if triggerCheck != nil {
			highlights := make(map[string]string)
			for _, highlight := range searchResults[triggerCheckInd].Highlights {
				highlights[highlight.Field] = highlight.Value
			}
			triggerCheck.Highlights = highlights
			triggersList.List = append(triggersList.List, *triggerCheck)
		}
	}

	return &triggersList, nil
}

func DeleteTriggersPager(dataBase moira.Database, pagerID string) (dto.TriggersSearchResultDeleteResponse, *api.ErrorResponse) {
	exists, err := dataBase.IsTriggersSearchResultsExist(pagerID)
	if err != nil {
		return dto.TriggersSearchResultDeleteResponse{}, api.ErrorInternalServer(err)
	}
	if !exists {
		return dto.TriggersSearchResultDeleteResponse{}, api.ErrorNotFound(fmt.Sprintf("pager with id %s not found", pagerID))
	}
	err = dataBase.DeleteTriggersSearchResults(pagerID)
	if err != nil {
		return dto.TriggersSearchResultDeleteResponse{}, api.ErrorInternalServer(err)
	}
	return dto.TriggersSearchResultDeleteResponse{PagerID: pagerID}, nil
}

func triggerExists(dataBase moira.Database, triggerID string) (bool, error) {
	_, err := dataBase.GetTrigger(triggerID)
	if err == database.ErrNil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
