package controller

import (
	"fmt"
	"sort"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetAllTagsAndSubscriptions get tags subscriptions and triggerIDs.
func GetAllTagsAndSubscriptions(database moira.Database, logger moira.Logger) (*dto.TagsStatistics, *api.ErrorResponse) {
	tagsNames, err := database.GetTagNames()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	tagsStatistics := dto.TagsStatistics{
		List: make([]dto.TagStatistics, 0, len(tagsNames)),
	}
	rch := make(chan *dto.TagStatistics, len(tagsNames))

	for _, tagName := range tagsNames {
		go func(tagName string) {
			tagStat := &dto.TagStatistics{
				Subscriptions: make([]moira.SubscriptionData, 0),
			}
			tagStat.TagName = tagName
			subscriptions, err := database.GetTagsSubscriptions([]string{tagName})
			if err != nil {
				logger.Error().
					Error(err).
					Msg("Failed to get tag's subscriptions")
				rch <- nil
			}
			for _, subscription := range subscriptions {
				if subscription != nil {
					tagStat.Subscriptions = append(tagStat.Subscriptions, *subscription)
				}
			}
			tagStat.Triggers, err = database.GetTagTriggerIDs(tagName)
			if err != nil {
				logger.Error().
					Error(err).
					Msg("Failed to get tag trigger IDs")
				rch <- nil
			}
			rch <- tagStat
		}(tagName)
	}

	for i := 0; i < len(tagsNames); i++ {
		if r := <-rch; r != nil {
			tagsStatistics.List = append(tagsStatistics.List, *r)
		}
	}
	return &tagsStatistics, nil
}

// GetAllTags gets all tag names.
func GetAllTags(database moira.Database) (*dto.TagsData, *api.ErrorResponse) {
	tagsNames, err := getTagNamesSorted(database)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	tagsData := &dto.TagsData{
		TagNames: tagsNames,
	}

	return tagsData, nil
}

func getTagNamesSorted(database moira.Database) ([]string, error) {
	tagsNames, err := database.GetTagNames()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(tagsNames, func(i, j int) bool { return strings.ToLower(tagsNames[i]) < strings.ToLower(tagsNames[j]) })
	return tagsNames, nil
}

// CreateTags create tags with tag names.
func CreateTags(database moira.Database, tags *dto.TagsData) *api.ErrorResponse {
	if err := database.CreateTags(tags.TagNames); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// RemoveTag deletes tag by name.
func RemoveTag(database moira.Database, tagName string) (*dto.MessageResponse, *api.ErrorResponse) {
	triggerIDs, err := database.GetTagTriggerIDs(tagName)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if len(triggerIDs) > 0 {
		return nil, api.ErrorInvalidRequest(fmt.Errorf("this tag is assigned to %v triggers. Remove tag from triggers first", len(triggerIDs)))
	}

	subscriptions, err := database.GetTagsSubscriptions([]string{tagName})
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	for _, s := range subscriptions {
		if s != nil {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("this tag is assigned to %v subscriptions. Remove tag from subscriptions first", len(subscriptions)))
		}
	}

	if err = database.RemoveTag(tagName); err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	return &dto.MessageResponse{Message: "tag deleted"}, nil
}
