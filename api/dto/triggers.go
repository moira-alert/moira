// nolint
package dto

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/templating"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/expression"
	metricSource "github.com/moira-alert/moira/metric_source"
)

var targetNameRegex = regexp.MustCompile("^t\\d+$")

var (
	// errBadAloneMetricName is used when any key in map TriggerModel.AloneMetric doesn't match targetNameRegex.
	errBadAloneMetricName = errors.New("alone metrics' target name must match the pattern: ^t\\d+$, for example: 't1'")

	// errTargetsRequired is returned when there is no targets in Trigger.
	errTargetsRequired = errors.New("targets are required")

	// errTagsRequired is returned when there is no tags in Trigger.
	errTagsRequired = errors.New("tags are required")

	// errTriggerNameRequired is returned when there is empty Name in Trigger.
	errTriggerNameRequired = errors.New("trigger name is required")

	// errAloneMetricTargetIndexOutOfRange is returned when target index is out of range. Example: if we have target "t1",
	// then "1" is a target index.
	errAloneMetricTargetIndexOutOfRange = errors.New("alone metrics target index should be in range from 1 to length of targets")

	// errAsteriskPatternNotAllowed is returned then one of Trigger.Patterns contain only "*".
	errAsteriskPatternNotAllowed = errors.New("pattern \"*\" is not allowed to use")

	// errNoAllowedDays is returned then all days disabled in moira.ScheduleData.
	errNoAllowedDays = errors.New("no allowed days in trigger schedule")
)

// TODO(litleleprikon): Remove after https://github.com/moira-alert/moira/issues/550 will be resolved.
const asteriskPattern = "*"

type TriggersList struct {
	Page  *int64               `json:"page,omitempty" format:"int64" extensions:"x-nullable"`
	Size  *int64               `json:"size,omitempty" format:"int64" extensions:"x-nullable"`
	Total *int64               `json:"total,omitempty" format:"int64" extensions:"x-nullable"`
	Pager *string              `json:"pager,omitempty" extensions:"x-nullable"`
	List  []moira.TriggerCheck `json:"list"`
}

func (*TriggersList) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

type Trigger struct {
	TriggerModel
	Throttling int64 `json:"throttling" example:"0" format:"int64"`
}

// TriggerModel is moira.Trigger api representation.
type TriggerModel struct {
	// Trigger unique ID
	ID string `json:"id" example:"292516ed-4924-4154-a62c-ebe312431fce"`
	// Trigger name
	Name string `json:"name" example:"Not enough disk space left"`
	// Description string
	Desc *string `json:"desc,omitempty" example:"check the size of /var/log" extensions:"x-nullable"`
	// Graphite-like targets: t1, t2, ...
	Targets []string `json:"targets" example:"devOps.my_server.hdd.freespace_mbytes"`
	// WARN threshold
	WarnValue *float64 `json:"warn_value" example:"500" extensions:"x-nullable"`
	// ERROR threshold
	ErrorValue *float64 `json:"error_value" example:"1000" extensions:"x-nullable"`
	// Could be: rising, falling, expression
	TriggerType string `json:"trigger_type" example:"rising"`
	// Set of tags to manipulate subscriptions
	Tags []string `json:"tags" example:"server,disk"`
	// When there are no metrics for trigger, Moira will switch metric to TTLState state after TTL seconds
	TTLState *moira.TTLState `json:"ttl_state,omitempty" example:"NODATA" extensions:"x-nullable"`
	// When there are no metrics for trigger, Moira will switch metric to TTLState state after TTL seconds
	TTL int64 `json:"ttl,omitempty" example:"600" format:"int64"`
	// Determines when Moira should monitor trigger
	Schedule *moira.ScheduleData `json:"sched,omitempty" extensions:"x-nullable"`
	// Used if you need more complex logic than provided by WARN/ERROR values
	Expression string `json:"expression" example:""`
	// Graphite patterns for trigger
	Patterns []string `json:"patterns" example:""`
	// Shows if trigger is remote (graphite-backend) based or stored inside Moira-Redis DB
	//
	// Deprecated: Use TriggerSource field instead
	IsRemote bool `json:"is_remote" example:"false"`
	// Shows the type of source from where the metrics are fetched
	TriggerSource moira.TriggerSource `json:"trigger_source" example:"graphite_local"`
	// Shows the exact cluster from where the metrics are fetched
	ClusterId moira.ClusterId `json:"cluster_id" example:"default"`
	// If true, first event NODATA → OK will be omitted
	MuteNewMetrics bool `json:"mute_new_metrics" example:"false"`
	// A list of targets that have only alone metrics
	AloneMetrics map[string]bool `json:"alone_metrics" example:"t1:true"`
	// Datetime when the trigger was created
	CreatedAt *time.Time `json:"created_at" extensions:"x-nullable"`
	// Datetime  when the trigger was updated
	UpdatedAt *time.Time `json:"updated_at" extensions:"x-nullable"`
	// Username who created trigger
	CreatedBy string `json:"created_by"`
	// Username who updated trigger
	UpdatedBy string `json:"updated_by"`
}

// ClusterKey returns cluster key composed of trigger source and cluster id associated with the trigger.
func (model *TriggerModel) ClusterKey() moira.ClusterKey {
	return moira.MakeClusterKey(model.TriggerSource, model.ClusterId)
}

// ToMoiraTrigger transforms TriggerModel to moira.Trigger.
func (model *TriggerModel) ToMoiraTrigger() *moira.Trigger {
	return &moira.Trigger{
		ID:             model.ID,
		Name:           model.Name,
		Desc:           model.Desc,
		Targets:        model.Targets,
		WarnValue:      model.WarnValue,
		ErrorValue:     model.ErrorValue,
		TriggerType:    model.TriggerType,
		Tags:           model.Tags,
		TTLState:       model.TTLState,
		TTL:            model.TTL,
		Schedule:       model.Schedule,
		Expression:     &model.Expression,
		Patterns:       model.Patterns,
		TriggerSource:  model.TriggerSource,
		ClusterId:      model.ClusterId,
		MuteNewMetrics: model.MuteNewMetrics,
		AloneMetrics:   model.AloneMetrics,
		UpdatedBy:      model.UpdatedBy,
	}
}

// CreateTriggerModel transforms moira.Trigger to TriggerModel.
func CreateTriggerModel(trigger *moira.Trigger) TriggerModel {
	return TriggerModel{
		ID:             trigger.ID,
		Name:           trigger.Name,
		Desc:           trigger.Desc,
		Targets:        trigger.Targets,
		WarnValue:      trigger.WarnValue,
		ErrorValue:     trigger.ErrorValue,
		TriggerType:    trigger.TriggerType,
		Tags:           trigger.Tags,
		TTLState:       trigger.TTLState,
		TTL:            trigger.TTL,
		Schedule:       trigger.Schedule,
		Expression:     moira.UseString(trigger.Expression),
		Patterns:       trigger.Patterns,
		IsRemote:       trigger.TriggerSource == moira.GraphiteRemote,
		TriggerSource:  trigger.TriggerSource,
		ClusterId:      trigger.ClusterId,
		MuteNewMetrics: trigger.MuteNewMetrics,
		AloneMetrics:   trigger.AloneMetrics,
		CreatedAt:      getDateTime(trigger.CreatedAt),
		UpdatedAt:      getDateTime(trigger.UpdatedAt),
		CreatedBy:      trigger.CreatedBy,
		UpdatedBy:      trigger.UpdatedBy,
	}
}

func (trigger *Trigger) Bind(request *http.Request) error {
	trigger.Tags = normalizeTags(trigger.Tags)
	if len(trigger.Targets) == 0 {
		return api.ErrInvalidRequestContent{ValidationError: errTargetsRequired}
	}

	if len(trigger.Tags) == 0 {
		return api.ErrInvalidRequestContent{ValidationError: errTagsRequired}
	}

	if trigger.Name == "" {
		return api.ErrInvalidRequestContent{ValidationError: errTriggerNameRequired}
	}

	limits := middleware.GetLimits(request)
	if utf8.RuneCountInString(trigger.Name) > limits.Trigger.MaxNameSize {
		return api.ErrInvalidRequestContent{
			ValidationError: fmt.Errorf("trigger name too long, should not be less than %d symbols", limits.Trigger.MaxNameSize),
		}
	}

	if err := checkWarnErrorExpression(trigger); err != nil {
		return api.ErrInvalidRequestContent{ValidationError: err}
	}

	if len(trigger.Targets) <= 1 { // we should have empty alone metrics dictionary when there is only one target
		trigger.AloneMetrics = map[string]bool{}
	}

	for targetName := range trigger.AloneMetrics {
		if !targetNameRegex.MatchString(targetName) {
			return api.ErrInvalidRequestContent{ValidationError: errBadAloneMetricName}
		}

		targetIndexStr := targetName[1:]
		targetIndex, err := strconv.Atoi(targetIndexStr)
		if err != nil {
			return api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("alone metrics target index should be valid number: %w", err)}
		}

		if targetIndex < 0 || targetIndex > len(trigger.Targets) {
			return api.ErrInvalidRequestContent{ValidationError: errAloneMetricTargetIndexOutOfRange}
		}
	}

	if trigger.TTLState == nil {
		trigger.TTLState = &moira.TTLStateNODATA
	}

	triggerExpression := expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64),
		WarnValue:               trigger.WarnValue,
		ErrorValue:              trigger.ErrorValue,
		TriggerType:             trigger.TriggerType,
		PreviousState:           moira.StateNODATA,
		Expression:              &trigger.Expression,
	}

	trigger.TriggerSource = trigger.TriggerSource.FillInIfNotSet(trigger.IsRemote)
	trigger.ClusterId = trigger.ClusterId.FillInIfNotSet()

	metricsSourceProvider := middleware.GetTriggerTargetsSourceProvider(request)
	metricsSource, err := metricsSourceProvider.GetMetricSource(trigger.ClusterKey())
	if err != nil {
		return err
	}

	if trigger.TTL == 0 {
		trigger.TTL = moira.DefaultTTL
	}
	if err := checkTTLSanity(trigger, metricsSource); err != nil {
		return api.ErrInvalidRequestContent{ValidationError: err}
	}

	metricsDataNames, err := resolvePatterns(trigger, &triggerExpression, metricsSource)
	if err != nil {
		return err
	}

	err = checkResolvedPatterns(trigger)
	if err != nil {
		return api.ErrInvalidRequestContent{ValidationError: err}
	}

	if trigger.Schedule == nil {
		trigger.Schedule = moira.NewDefaultScheduleData()
	} else {
		correctedSchedule, err := checkScheduleFilling(trigger.Schedule)
		if err != nil {
			return api.ErrInvalidRequestContent{ValidationError: err}
		}

		trigger.Schedule = correctedSchedule
	}

	middleware.SetTimeSeriesNames(request, metricsDataNames)

	if err = triggerExpression.Validate(); err != nil {
		return err
	}

	trigger.UpdatedBy = middleware.GetLogin(request)

	return nil
}

func getDateTime(timestamp *int64) *time.Time {
	if timestamp == nil {
		return nil
	}

	datetime := time.Unix(*timestamp, 0).UTC()

	return &datetime
}

// checkScheduleFilling ensures that all days are included to schedule, ordered from monday to sunday
// and have proper names (one of [Mon, Tue, Wed, Thu, Fri, Sat Sun]).
func checkScheduleFilling(gotSchedule *moira.ScheduleData) (*moira.ScheduleData, error) {
	newSchedule := moira.NewDefaultScheduleData()

	scheduleDaysMap := make(map[moira.DayName]bool, len(newSchedule.Days))
	for _, day := range newSchedule.Days {
		scheduleDaysMap[day.Name] = false
	}

	badDayNames := make([]string, 0)
	for _, day := range gotSchedule.Days {
		_, validDayName := scheduleDaysMap[day.Name]
		if validDayName {
			scheduleDaysMap[day.Name] = day.Enabled
		} else {
			badDayNames = append(badDayNames, string(day.Name))
		}
	}

	if len(badDayNames) != 0 {
		return nil, fmt.Errorf("bad day names in schedule: %s", strings.Join(badDayNames, ", "))
	}

	someDayEnabled := false
	for i := range newSchedule.Days {
		newSchedule.Days[i].Enabled = scheduleDaysMap[newSchedule.Days[i].Name]

		if newSchedule.Days[i].Enabled {
			someDayEnabled = true
		}
	}

	if !someDayEnabled {
		return nil, errNoAllowedDays
	}

	newSchedule.TimezoneOffset = gotSchedule.TimezoneOffset
	newSchedule.StartOffset = gotSchedule.StartOffset
	newSchedule.EndOffset = gotSchedule.EndOffset

	return newSchedule, nil
}

func checkResolvedPatterns(trigger *Trigger) error {
	for _, pattern := range trigger.Patterns {
		// TODO(litleleprikon): Remove after https://github.com/moira-alert/moira/issues/550 will be resolved
		if pattern == asteriskPattern {
			return errAsteriskPatternNotAllowed
		}

		err := checkRegexpInPattern(pattern)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkRegexpInPattern(pattern string) error {
	if !strings.HasPrefix(pattern, "seriesByTag") {
		return nil
	}

	tagSpecs, err := filter.ParseSeriesByTag(pattern)
	if err != nil {
		return err
	}

	for _, spec := range tagSpecs {
		if spec.Operator == filter.MatchOperator || spec.Operator == filter.NotMatchOperator {
			_, err = regexp.Compile(spec.Value)
			if err != nil {
				return fmt.Errorf("bad regexp in tag '%s': %w", spec.Name, err)
			}
		}
	}

	return nil
}

func checkTTLSanity(trigger *Trigger, metricsSource metricSource.MetricSource) error {
	maximumAllowedTTL := metricsSource.GetMetricsTTLSeconds()

	if trigger.TTL > maximumAllowedTTL {
		var triggerType string

		switch trigger.TriggerSource {
		case moira.GraphiteLocal:
			triggerType = "graphite local"

		case moira.GraphiteRemote:
			triggerType = "graphite remote"

		case moira.PrometheusRemote:
			triggerType = "prometheus remote"
		}

		return fmt.Errorf("TTL for %s trigger can't be more than %d seconds", triggerType, maximumAllowedTTL)
	}
	return nil
}

func resolvePatterns(trigger *Trigger, expressionValues *expression.TriggerExpression, metricsSource metricSource.MetricSource) (map[string]bool, error) {
	now := time.Now().Unix()
	targetNum := 1
	trigger.Patterns = make([]string, 0)
	metricsDataNames := make(map[string]bool)

	for _, tar := range trigger.Targets {
		fetchResult, err := metricsSource.Fetch(tar, now-600, now, false)
		if err != nil {
			return nil, err
		}
		targetPatterns, err := fetchResult.GetPatterns()
		if err == nil {
			trigger.Patterns = append(trigger.Patterns, targetPatterns...)
		}

		if targetNum == 1 {
			expressionValues.MainTargetValue = 42
		} else {
			targetName := fmt.Sprintf("t%v", targetNum)
			expressionValues.AdditionalTargetsValues[targetName] = 42
		}
		for _, metricData := range fetchResult.GetMetricsData() {
			metricsDataNames[metricData.Name] = true
		}
		targetNum++
	}
	return metricsDataNames, nil
}

func checkWarnErrorExpression(trigger *Trigger) error {
	if trigger.WarnValue == nil && trigger.ErrorValue == nil && trigger.Expression == "" {
		return fmt.Errorf("at least one of error_value, warn_value or expression is required")
	}

	if trigger.WarnValue != nil && trigger.ErrorValue != nil && *trigger.WarnValue == *trigger.ErrorValue {
		return fmt.Errorf("error_value is equal to warn_value, please set exactly one value")
	}

	switch trigger.TriggerType {
	case "":
		if trigger.Expression != "" {
			trigger.TriggerType = moira.ExpressionTrigger
			return nil
		}
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			if *trigger.WarnValue > *trigger.ErrorValue {
				trigger.TriggerType = moira.FallingTrigger
				return nil
			}
			if *trigger.WarnValue < *trigger.ErrorValue {
				trigger.TriggerType = moira.RisingTrigger
				return nil
			}
		}
		if trigger.WarnValue == nil {
			return fmt.Errorf("warn_value: is empty - please fill both values or choose trigger_type: rising, falling, expression")
		}
		if trigger.ErrorValue == nil {
			return fmt.Errorf("error_value: is empty - please fill both values or choose trigger_type: rising, falling, expression")
		}

	case moira.RisingTrigger:
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			if *trigger.WarnValue > *trigger.ErrorValue {
				return fmt.Errorf("error_value should be greater than warn_value")
			}
		}
		if err := checkSimpleModeFields(trigger); err != nil {
			return err
		}

	case moira.FallingTrigger:
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			if *trigger.WarnValue < *trigger.ErrorValue {
				return fmt.Errorf("warn_value should be greater than error_value")
			}
		}
		if err := checkSimpleModeFields(trigger); err != nil {
			return err
		}

	case moira.ExpressionTrigger:
		if trigger.Expression == "" {
			return fmt.Errorf("trigger_type set to expression, but no expression provided")
		}
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			return fmt.Errorf("can't use 'warn_value' and 'error_value' on trigger_type: '%v'", moira.ExpressionTrigger)
		}
		if trigger.WarnValue != nil {
			return fmt.Errorf("can't use 'warn_value' on trigger_type: '%v'", moira.ExpressionTrigger)
		}
		if trigger.ErrorValue != nil {
			return fmt.Errorf("can't use 'error_value' on trigger_type: '%v'", moira.ExpressionTrigger)
		}

	default:
		return fmt.Errorf("wrong trigger_type: %v, allowable values: '%v', '%v', '%v'",
			trigger.TriggerType, moira.RisingTrigger, moira.FallingTrigger, moira.ExpressionTrigger)
	}

	return nil
}

func checkSimpleModeFields(trigger *Trigger) error {
	if len(trigger.Targets) > 1 {
		return fmt.Errorf("can't use trigger_type not '%v' for with multiple targets", trigger.TriggerType)
	}
	if trigger.Expression != "" {
		return fmt.Errorf("can't use 'expression' to trigger_type: '%v'", trigger.TriggerType)
	}
	return nil
}

func (*Trigger) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

// PopulatedDescription returns new trigger description after template populating.
func (trigger *Trigger) PopulatedDescription(events moira.NotificationEvents) (*string, error) {
	emptyString := ""

	if trigger.Desc == nil {
		return &emptyString, nil
	}

	triggerDescriptionPopulater := templating.NewTriggerDescriptionPopulater(trigger.Name, events.ToTemplateEvents())
	newDescription, err := triggerDescriptionPopulater.Populate(*trigger.Desc)
	if err != nil {
		return &emptyString, fmt.Errorf("you have an error in your Go template: %v", err)
	}

	return &newDescription, nil
}

type TriggerCheckResponse struct {
	// Graphite-like targets: t1, t2, ...
	Targets []TreeOfProblems `json:"targets,omitempty"`
}

type TriggerCheck struct {
	*moira.CheckData
	TriggerID string `json:"trigger_id" example:"trigger_id"`
}

func (*TriggerCheck) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

type MetricsMaintenance map[string]int64

func (*MetricsMaintenance) Bind(*http.Request) error {
	return nil
}

type TriggerMaintenance struct {
	Trigger *int64           `json:"trigger" example:"1594225165" format:"int64" extensions:"x-nullable"`
	Metrics map[string]int64 `json:"metrics"`
}

func (*TriggerMaintenance) Bind(*http.Request) error {
	return nil
}

type ThrottlingResponse struct {
	Throttling int64 `json:"throttling" example:"0" format:"int64"`
}

func (*ThrottlingResponse) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

type SaveTriggerResponse struct {
	ID          string               `json:"id" example:"trigger_id"`
	Message     string               `json:"message" example:"trigger created"`
	CheckResult TriggerCheckResponse `json:"checkResult,omitempty"`
}

func (*SaveTriggerResponse) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

type TriggerMetrics map[string]map[string][]moira.MetricValue

func (*TriggerMetrics) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

type PatternMetrics struct {
	Pattern    string                          `json:"pattern"`
	Metrics    map[string][]*moira.MetricValue `json:"metrics"`
	Retentions map[string]int64                `json:"retention"`
}

type TriggerDump struct {
	Created   string           `json:"created"`
	LastCheck moira.CheckData  `json:"last_check"`
	Trigger   moira.Trigger    `json:"trigger"`
	Metrics   []PatternMetrics `json:"metrics"`
}

type TriggersSearchResultDeleteResponse struct {
	PagerID string `json:"pager_id" example:"292516ed-4924-4154-a62c-ebe312431fce"`
}

func (TriggersSearchResultDeleteResponse) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

// TriggerNoisiness represents TriggerCheck with amount of events for this trigger.
type TriggerNoisiness struct {
	Trigger
	// EventsCount for the trigger.
	EventsCount int64 `json:"events_count"`
}

func (*TriggerNoisiness) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

// TriggerNoisinessList represents list of TriggerNoisiness.
type TriggerNoisinessList ListDTO[*TriggerNoisiness]

func (*TriggerNoisinessList) Render(http.ResponseWriter, *http.Request) error {
	return nil
}
