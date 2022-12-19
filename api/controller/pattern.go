package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetAllPatterns get all patterns and triggers and metrics info corresponding to this pattern
func GetAllPatterns(database moira.Database, logger moira.Logger) (*dto.PatternList, *api.ErrorResponse) {
	patterns, err := database.GetPatterns()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	pattersList := dto.PatternList{
		List: make([]dto.PatternData, 0, len(patterns)),
	}

	rch := make(chan *dto.PatternData, len(patterns))

	for _, pattern := range patterns {
		go func(pattern string) {
			triggerIDs, err := database.GetPatternTriggerIDs(pattern)
			if err != nil {
				logger.Errorb().
					Error(err).
					Msg("Failed to get pattern trigger IDs")
				rch <- nil
			}
			triggers, err := database.GetTriggers(triggerIDs)
			if err != nil {
				logger.Errorb().
					Error(err).
					Msg("Failed to get trigger")
				rch <- nil
			}
			metrics, err := database.GetPatternMetrics(pattern)
			if err != nil {
				logger.Errorb().
					Error(err).
					Msg("Failed to get pattern metrics")
				rch <- nil
			}
			patternData := dto.PatternData{
				Pattern:  pattern,
				Triggers: make([]dto.TriggerModel, 0),
				Metrics:  metrics,
			}
			for _, trigger := range triggers {
				if trigger != nil {
					patternData.Triggers = append(patternData.Triggers, dto.CreateTriggerModel(trigger))
				}
			}
			rch <- &patternData
		}(pattern)
	}

	for i := 0; i < len(patterns); i++ {
		if r := <-rch; r != nil {
			pattersList.List = append(pattersList.List, *r)
		}
	}

	return &pattersList, nil
}

// DeletePattern deletes trigger pattern
func DeletePattern(database moira.Database, pattern string) *api.ErrorResponse {
	if err := database.RemovePattern(pattern); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
