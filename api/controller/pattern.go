package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
)

func GetAllPatterns(database moira.Database) (*dto.PatternList, *dto.ErrorResponse) {
	//todo разница в 7 строк, разобраться
	//todo работает медлено
	patterns, err := database.GetPatterns()
	pattersList := dto.PatternList{
		List: make([]dto.Pattern, 0, len(patterns)),
	}
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	for _, pattern := range patterns {
		triggerIds, err := database.GetPatternTriggerIds(pattern)
		if err != nil {
			return nil, dto.ErrorInternalServer(err)
		}
		triggersList, err := database.GetTriggers(triggerIds)
		if err != nil {
			return nil, dto.ErrorInternalServer(err)
		}
		metrics, err := database.GetPatternMetrics(pattern)
		if err != nil {
			return nil, dto.ErrorInternalServer(err)
		}
		pattersList.List = append(pattersList.List, dto.Pattern{Pattern: pattern, Triggers: triggersList, Metrics: metrics})

	}
	return &pattersList, nil
}

func DeletePattern(database moira.Database, pattern string) *dto.ErrorResponse {
	if err := database.RemovePattern(pattern); err != nil {
		return dto.ErrorInternalServer(err)
	}
	return nil
}
