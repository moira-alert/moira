package controller

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
)

func GetAllTagsAndSubscriptions(database moira.Database) (*dto.TagsStatistics, *dto.ErrorResponse) {
	//todo разница в 1 строку, разобраться
	//todo работает медлено
	tagsNames, err := database.GetTagNames()
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	tagsStatistics := dto.TagsStatistics{
		List: make([]dto.TagStatistics, 0, len(tagsNames)),
	}

	for _, tagName := range tagsNames {
		tagStat := dto.TagStatistics{}
		tagStat.TagName = tagName
		tagStat.Subscriptions, err = database.GetTagsSubscriptions([]string{tagName})
		if err != nil {
			return nil, dto.ErrorInternalServer(err)
		}
		tagStat.Triggers, err = database.GetTagTriggerIds(tagName)
		if err != nil {
			return nil, dto.ErrorInternalServer(err)
		}
		tagStat.Data, err = database.GetTag(tagName)
		if err != nil {
			return nil, dto.ErrorInternalServer(err)
		}
		tagsStatistics.List = append(tagsStatistics.List, tagStat)
	}
	return &tagsStatistics, nil
}

func GetAllTags(database moira.Database) (*dto.TagsData, *dto.ErrorResponse) {
	tagsNames, err := database.GetTagNames()
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	tagsMap, err := database.GetTags(tagsNames)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	tagsData := &dto.TagsData{
		TagNames: tagsNames,
		TagsMap:  tagsMap,
	}

	return tagsData, nil
}

func DeleteTag(database moira.Database, tagName string) (*dto.MessageResponse, *dto.ErrorResponse) {
	triggerIds, err := database.GetTagTriggerIds(tagName)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	if len(triggerIds) > 0 {
		return nil, dto.ErrorInvalidRequest(fmt.Errorf("This tag is assigned to %v triggers. Remove tag from triggers first", len(triggerIds)))
	} else {
		if err = database.DeleteTag(tagName); err != nil {
			return nil, dto.ErrorInternalServer(err)
		}
	}
	return &dto.MessageResponse{Message: "tag deleted"}, nil
}

func SetTagMaintenance(database moira.Database, tagName string, tag *dto.Tag) *dto.ErrorResponse {
	data := moira.TagData(*tag)

	if err := database.SetTagMaintenance(tagName, data); err != nil {
		return dto.ErrorInternalServer(err)
	}

	return nil
}
