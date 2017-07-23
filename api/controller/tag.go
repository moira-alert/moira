package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
)

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

func SetTagMaintenance(database moira.Database, tagName string, tag *dto.Tag) *dto.ErrorResponse {
	data := moira.TagData(*tag)

	if err := database.SetTagMaintenance(tagName, data); err != nil {
		return dto.ErrorInternalServer(err)
	}

	return nil
}
