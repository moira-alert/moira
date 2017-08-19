package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
)

//GetAllPatterns get all patterns and triggers and metrics info corresponding to this pattern
func GetAllPatterns(database moira.Database) (*dto.PatternList, *api.ErrorResponse) {
	//todo работает медлено
	patterns, err := database.GetPatterns()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	pattersList := dto.PatternList{
		List: make([]dto.PatternData, 0, len(patterns)),
	}

	for _, pattern := range patterns {
		triggerIDs, err := database.GetPatternTriggerIds(pattern)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		triggersList, err := database.GetTriggers(triggerIDs)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		metrics, err := database.GetPatternMetrics(pattern)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		pattersList.List = append(pattersList.List, dto.PatternData{Pattern: pattern, Triggers: triggersList, Metrics: metrics})

	}
	return &pattersList, nil
}

//DeletePattern deletes trigger pattern
func DeletePattern(database moira.Database, pattern string) *api.ErrorResponse {
	if err := database.RemovePattern(pattern); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
