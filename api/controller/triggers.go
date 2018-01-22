package controller

import (
	"fmt"

	"github.com/satori/go.uuid"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
)

// CreateTrigger creates new trigger
func CreateTrigger(dataBase moira.Database, trigger *dto.TriggerModel, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	if trigger.ID == "" {
		trigger.ID = uuid.NewV4().String()
	} else {
		exists, err := isTriggerExists(dataBase, trigger.ID)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		if exists {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("Trigger with this ID already exists"))
		}
	}
	if err := checkTriggerTags(trigger.Tags); err != nil {
		return nil, api.ErrorInvalidRequest(err)
	}
	resp, err := saveTrigger(dataBase, trigger.ToMoiraTrigger(), trigger.ID, timeSeriesNames)
	if resp != nil {
		resp.Message = "trigger created"
	}
	return resp, err
}

func isTriggerExists(dataBase moira.Database, triggerID string) (bool, error) {
	_, err := dataBase.GetTrigger(triggerID)
	if err == database.ErrNil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetAllTriggers gets all moira triggers
func GetAllTriggers(database moira.Database) (*dto.TriggersList, *api.ErrorResponse) {
	triggerIDs, err := database.GetTriggerIDs()
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

// GetTriggerPage gets trigger page and filter trigger by tags and errors
func GetTriggerPage(database moira.Database, page int64, size int64, onlyErrors bool, filterTags []string) (*dto.TriggersList, *api.ErrorResponse) {
	triggerIDs, err := database.GetTriggerCheckIDs(filterTags, onlyErrors)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	total := int64(len(triggerIDs))
	triggerIDs = getTriggerIdsRange(triggerIDs, total, page, size)
	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggersList := dto.TriggersList{
		List:  make([]moira.TriggerCheck, 0),
		Total: &total,
		Page:  &page,
		Size:  &size,
	}

	for _, triggerCheck := range triggerChecks {
		if triggerCheck != nil {
			triggersList.List = append(triggersList.List, *triggerCheck)
		}
	}
	return &triggersList, nil
}

func getTriggerIdsRange(triggerIDs []string, total int64, page int64, size int64) []string {
	from := page * size
	to := (page + 1) * size

	if from > total {
		from = total
	}

	if to > total {
		to = total
	}

	return triggerIDs[from:to]
}

func checkTriggerTags(tags []string) error {
	for _, tag := range tags {
		switch tag {
		case moira.EventHighDegradationTag, moira.EventDegradationTag, moira.EventProgressTag:
			return fmt.Errorf("Can't use reserved keyword: %s", tag)
		}
	}
	return nil
}
