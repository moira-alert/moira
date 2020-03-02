package controller

import (
	"fmt"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/gofrs/uuid"

	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	"github.com/moira-alert/moira/internal/database"
)

// CreateTrigger creates new trigger
func CreateTrigger(dataBase moira2.Database, trigger *dto.TriggerModel, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	if trigger.ID == "" {
		uuid4, err := uuid.NewV4()
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		trigger.ID = uuid4.String()
	} else {
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
func GetAllTriggers(database moira2.Database) (*dto.TriggersList, *api.ErrorResponse) {
	triggerIDs, err := database.GetAllTriggerIDs()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggersList := dto.TriggersList{
		List: make([]moira2.TriggerCheck, 0),
	}
	for _, triggerCheck := range triggerChecks {
		if triggerCheck != nil {
			triggersList.List = append(triggersList.List, *triggerCheck)
		}
	}
	return &triggersList, nil
}

// SearchTriggers gets trigger page and filter trigger by tags and search request terms
func SearchTriggers(database moira2.Database, searcher moira2.Searcher, page int64, size int64, onlyErrors bool, filterTags []string, searchString string) (*dto.TriggersList, *api.ErrorResponse) {
	searchResults, total, err := searcher.SearchTriggers(filterTags, searchString, onlyErrors, page, size)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	var triggerIDs []string
	for _, searchResult := range searchResults {
		triggerIDs = append(triggerIDs, searchResult.ObjectID)
	}

	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggersList := dto.TriggersList{
		List:  make([]moira2.TriggerCheck, 0),
		Total: &total,
		Page:  &page,
		Size:  &size,
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

func triggerExists(dataBase moira2.Database, triggerID string) (bool, error) {
	_, err := dataBase.GetTrigger(triggerID)
	if err == database.ErrNil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
