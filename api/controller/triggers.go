package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/satori/go.uuid"
)

//CreateTrigger creates new trigger
func CreateTrigger(database moira.Database, trigger *moira.Trigger, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	triggerID := uuid.NewV4().String()
	resp, err := SaveTrigger(database, trigger, triggerID, timeSeriesNames)
	if resp != nil {
		resp.Message = "trigger created"
	}
	return resp, err
}

//GetAllTriggers gets all moira triggers
func GetAllTriggers(database moira.Database) (*dto.TriggersList, *api.ErrorResponse) {
	triggerIDs, err := database.GetTriggerIds()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggersList := dto.TriggersList{
		List: triggerChecks,
	}
	return &triggersList, nil
}

//GetTriggerPage gets trigger page and filter trigger by tags and errors
func GetTriggerPage(database moira.Database, page int64, size int64, onlyErrors bool, filterTags []string) (*dto.TriggersList, *api.ErrorResponse) {
	var triggersChecks []moira.TriggerChecks
	var total int64
	var err error

	if !onlyErrors && len(filterTags) == 0 {
		triggersChecks, total, err = getNotFilteredTriggers(database, page, size)
	} else {
		triggersChecks, total, err = getFilteredTriggers(database, page, size, onlyErrors, filterTags)
	}
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	triggersList := dto.TriggersList{
		List:  triggersChecks,
		Total: &total,
		Page:  &page,
		Size:  &size,
	}
	return &triggersList, nil
}

func getNotFilteredTriggers(database moira.Database, page int64, size int64) ([]moira.TriggerChecks, int64, error) {
	triggerIDs, total, err := database.GetTriggerCheckIDs()
	if err != nil {
		return nil, 0, err
	}
	triggerIDs = getTriggerIdsRange(triggerIDs, total, page, size)
	triggersChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, 0, err
	}
	return triggersChecks, total, nil
}

func getFilteredTriggers(database moira.Database, page int64, size int64, onlyErrors bool, filterTags []string) ([]moira.TriggerChecks, int64, error) {
	triggerIDs, total, err := database.GetFilteredTriggerCheckIds(filterTags, onlyErrors)
	if err != nil {
		return nil, 0, err
	}
	triggerIDs = getTriggerIdsRange(triggerIDs, total, page, size)
	triggersChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, 0, err
	}
	return triggersChecks, total, nil
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
