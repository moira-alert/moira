package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
)

func GetAllTriggers(database moira.Database) (*dto.TriggersList, *dto.ErrorResponse) {
	//todo разница в 8 строк, разобраться
	triggersIds, err := database.GetTriggerIds()
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}
	triggerChecks, err := database.GetTriggerChecks(triggersIds)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}
	triggersList := dto.TriggersList{
		List: triggerChecks,
	}
	return &triggersList, nil
}

func GetTriggerPage(database moira.Database, page int64, size int64, onlyErrors bool, filterTags []string) (*dto.TriggersList, *dto.ErrorResponse) {
	var triggersChecks []moira.TriggerChecks
	var total int64
	var err error

	if !onlyErrors && len(filterTags) == 0 {
		triggersChecks, total, err = getNotFilteredTriggers(database, page, size)
	} else {
		triggersChecks, total, err = getFilteredTriggers(database, page, size, onlyErrors, filterTags)
	}
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}
	//todo Выпилить лишние поля из JSON'a
	triggersList := dto.TriggersList{
		List:  triggersChecks,
		Total: &total,
		Page:  &page,
		Size:  &size,
	}
	return &triggersList, nil
}

func getNotFilteredTriggers(database moira.Database, page int64, size int64) ([]moira.TriggerChecks, int64, error) {
	triggerIds, total, err := database.GetTriggerCheckIds()
	if err != nil {
		return nil, 0, err
	}
	triggerIds = triggerIds[page*size : (page+1)*size]
	triggersChecks, err := database.GetTriggerChecks(triggerIds)
	if err != nil {
		return nil, 0, err
	}
	return triggersChecks, total, nil
}

func getFilteredTriggers(database moira.Database, page int64, size int64, onlyErrors bool, filterTags []string) ([]moira.TriggerChecks, int64, error) {
	triggerIds, total, err := database.GetFilteredTriggerCheckIds(filterTags, onlyErrors)
	if err != nil {
		return nil, 0, err
	}

	from := page * size
	to := (page + 1) * size

	if from > total {
		from = total
	}

	if to > total {
		to = total
	}

	triggerIds = triggerIds[from:to]
	triggersChecks, err := database.GetTriggerChecks(triggerIds)
	if err != nil {
		return nil, 0, err
	}
	return triggersChecks, total, nil
}
