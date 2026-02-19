package controller

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	db "github.com/moira-alert/moira/database"
)

const pageSizeUnlimited int64 = -1

var (
	idValidationPatternString string         = `^[A-Za-z0-9._~-]*$`
	idValidationPattern       *regexp.Regexp = regexp.MustCompile(idValidationPatternString)
	teamIDVaildationErrorMsg  string         = fmt.Sprintf("team ID contains invalid characters that do not match the pattern: %s", idValidationPatternString)
)

// CreateTrigger creates new trigger.
func CreateTrigger(dataBase moira.Database, trigger *dto.TriggerModel, timeSeriesNames map[string]bool) (*dto.SaveTriggerResponse, *api.ErrorResponse) {
	if trigger.ID == "" {
		uuid4, err := uuid.NewV4()
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}

		trigger.ID = uuid4.String()
	} else {
		if !idValidationPattern.MatchString(trigger.ID) {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger ID contains invalid characters (allowed: 0-9, a-z, A-Z, -, ~, _, .)"))
		}

		exists, err := triggerExists(dataBase, trigger.ID)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}

		if exists {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger with this ID (%s) already exists", trigger.ID))
		}
	}

	if !isTeamIDValid(trigger.TeamID) {
		return nil, api.ErrorInvalidRequest(fmt.Errorf(teamIDVaildationErrorMsg))
	}

	resp, err := saveTrigger(dataBase, nil, trigger.ToMoiraTrigger(), trigger.ID, timeSeriesNames)
	if resp != nil {
		resp.Message = "trigger created"
	}

	return resp, err
}

// GetAllTriggers gets all moira triggers.
func GetAllTriggers(database moira.Database) (*dto.TriggersList, *api.ErrorResponse) {
	triggerIDs, err := database.GetAllTriggerIDs()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	triggerChecks, err := getTriggerChecks(database, triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	triggersList := &dto.TriggersList{
		List: triggerChecks,
	}

	return triggersList, nil
}

// SearchTriggers gets trigger page and filter trigger by tags and search request terms.
func SearchTriggers(database moira.Database, searcher moira.Searcher, options moira.SearchOptions) (*dto.TriggersList, *api.ErrorResponse) { //nolint
	var searchResults []*moira.SearchResult

	var total int64

	pagerShouldExist := options.PagerID != ""

	if pagerShouldExist && (options.SearchString != "" || len(options.Tags) > 0) {
		return nil, api.ErrorInvalidRequest(fmt.Errorf("cannot handle request with search string or tags and pager ID set"))
	}

	if pagerShouldExist {
		var err error

		searchResults, total, err = database.GetTriggersSearchResults(options.PagerID, options.Page, options.Size)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}

		if searchResults == nil {
			return nil, api.ErrorNotFound("Pager not found")
		}
	} else {
		var err error

		if options.CreatePager {
			options.Size = pageSizeUnlimited
		}

		searchResults, total, err = searcher.SearchTriggers(options)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
	}

	if options.CreatePager && !pagerShouldExist {
		uuid4, err := uuid.NewV4()
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}

		options.PagerID = uuid4.String()

		err = database.SaveTriggersSearchResults(options.PagerID, searchResults, options.PagerTTL)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
	}

	if options.CreatePager {
		var from, to int64 = 0, int64(len(searchResults))
		if options.Size >= 0 {
			from = int64(math.Min(float64(options.Page*options.Size), float64(len(searchResults))))
			to = int64(math.Min(float64(from+options.Size), float64(len(searchResults))))
		}

		searchResults = searchResults[from:to]
	}

	triggerIDs := make([]string, 0, len(searchResults))
	for _, searchResult := range searchResults {
		triggerIDs = append(triggerIDs, searchResult.ObjectID)
	}

	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	var pagerIDPtr *string
	if options.PagerID != "" {
		pagerIDPtr = &options.PagerID
	}

	triggersList := dto.TriggersList{
		List:  make([]moira.TriggerCheck, 0),
		Total: &total,
		Page:  &options.Page,
		Size:  &options.Size,
		Pager: pagerIDPtr,
	}

	for triggerCheckInd := range triggerChecks {
		triggerCheck := triggerChecks[triggerCheckInd]
		if triggerCheck != nil {
			triggerCheck.LastCheck.RemoveDeadMetrics()

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

func DeleteTriggersPager(database moira.Database, pagerID string) (dto.TriggersSearchResultDeleteResponse, *api.ErrorResponse) {
	exists, err := database.IsTriggersSearchResultsExist(pagerID)
	if err != nil {
		return dto.TriggersSearchResultDeleteResponse{}, api.ErrorInternalServer(err)
	}

	if !exists {
		return dto.TriggersSearchResultDeleteResponse{}, api.ErrorNotFound(fmt.Sprintf("pager with id %s not found", pagerID))
	}

	err = database.DeleteTriggersSearchResults(pagerID)
	if err != nil {
		return dto.TriggersSearchResultDeleteResponse{}, api.ErrorInternalServer(err)
	}

	return dto.TriggersSearchResultDeleteResponse{PagerID: pagerID}, nil
}

// GetUnusedTriggerIDs returns unused triggers ids.
func GetUnusedTriggerIDs(database moira.Database) (*dto.TriggersList, *api.ErrorResponse) {
	triggerIDs, err := database.GetUnusedTriggerIDs()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	triggerChecks, err := getTriggerChecks(database, triggerIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	triggersList := &dto.TriggersList{
		List: triggerChecks,
	}

	return triggersList, nil
}

func getTriggerChecks(database moira.Database, triggerIDs []string) ([]moira.TriggerCheck, error) {
	triggerChecks, err := database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return nil, err
	}

	list := make([]moira.TriggerCheck, 0, len(triggerChecks))

	for _, triggerCheck := range triggerChecks {
		if triggerCheck != nil {
			triggerCheck.LastCheck.RemoveDeadMetrics()
			list = append(list, *triggerCheck)
		}
	}

	return list, nil
}

func triggerExists(database moira.Database, triggerID string) (bool, error) {
	_, err := database.GetTrigger(triggerID)
	if errors.Is(err, db.ErrNil) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// GetTriggerNoisiness get triggers with amount of events (within time range [from, to])
// and sorts by events_count according to sortOrder.
func GetTriggerNoisiness(
	database moira.Database,
	page, size int64,
	from, to string,
	sortOrder api.SortOrder,
) (*dto.TriggerNoisinessList, *api.ErrorResponse) {
	triggerIDs, err := database.GetAllTriggerIDs()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	triggerIDsWithEventsCount := getTriggerIDsWithEventsCount(database, triggerIDs, from, to)

	sortTriggerIDsByEventsCount(triggerIDsWithEventsCount, sortOrder)

	total := int64(len(triggerIDsWithEventsCount))

	resDto := dto.TriggerNoisinessList{
		List:  []*dto.TriggerNoisiness{},
		Page:  page,
		Size:  size,
		Total: total,
	}

	triggerIDsWithEventsCount = applyPagination[triggerIDWithEventsCount](page, size, total, triggerIDsWithEventsCount)
	if len(triggerIDsWithEventsCount) == 0 {
		return &resDto, nil
	}

	triggers, err := getTriggerChecks(database, onlyTriggerIDs(triggerIDsWithEventsCount))
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	if len(triggers) != len(triggerIDsWithEventsCount) {
		return nil, api.ErrorInternalServer(fmt.Errorf("failed to fetch triggers for such range"))
	}

	resDto.List = make([]*dto.TriggerNoisiness, 0, len(triggers))
	for i := range triggers {
		resDto.List = append(resDto.List, &dto.TriggerNoisiness{
			Trigger: dto.Trigger{
				TriggerModel: dto.CreateTriggerModel(&triggers[i].Trigger),
				Throttling:   triggers[i].Throttling,
			},
			EventsCount: triggerIDsWithEventsCount[i].eventsCount,
		})
	}

	return &resDto, nil
}

type triggerIDWithEventsCount struct {
	triggerID   string
	eventsCount int64
}

func getTriggerIDsWithEventsCount(
	database moira.Database,
	triggerIDs []string,
	from, to string,
) []triggerIDWithEventsCount {
	resultTriggerIDs := make([]triggerIDWithEventsCount, 0, len(triggerIDs))

	for _, triggerID := range triggerIDs {
		eventsCount := database.GetNotificationEventCount(triggerID, from, to)
		resultTriggerIDs = append(resultTriggerIDs, triggerIDWithEventsCount{
			triggerID:   triggerID,
			eventsCount: eventsCount,
		})
	}

	return resultTriggerIDs
}

func sortTriggerIDsByEventsCount(idsWithCount []triggerIDWithEventsCount, sortOrder api.SortOrder) {
	if sortOrder == api.AscSortOrder || sortOrder == api.DescSortOrder {
		slices.SortFunc(idsWithCount, func(first, second triggerIDWithEventsCount) int {
			cmpRes := first.eventsCount - second.eventsCount

			if cmpRes == 0 {
				return strings.Compare(first.triggerID, second.triggerID)
			}

			if sortOrder == api.DescSortOrder {
				cmpRes *= -1
			}

			return int(cmpRes)
		})
	}
}

func onlyTriggerIDs(idsWithCount []triggerIDWithEventsCount) []string {
	triggerIDs := make([]string, 0, len(idsWithCount))

	for _, idWithCount := range idsWithCount {
		triggerIDs = append(triggerIDs, idWithCount.triggerID)
	}

	return triggerIDs
}

func isTeamIDValid(teamID string) bool {
	return teamID == "" || idValidationPattern.MatchString(teamID)
}
