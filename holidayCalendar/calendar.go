package holidayCalendar

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"time"

	ics "github.com/arran4/golang-ical"
)

// Config defines optional top-level holiday_calendar configuration section.
type Config struct {
	Enabled          bool     `yaml:"enabled"`
	File             string   `yaml:"file"`
	ExcludeSummaries []string `yaml:"exclude_summaries,omitempty"`
}

// Calendar checks holidays in a fixed timezone.
type Calendar struct {
	days map[string]struct{}
}

// New creates a holiday calendar.
func New(config Config) (*Calendar, error) {
	if !config.Enabled {
		return nil, nil
	}

	data, err := readCalendarFile(config.File)
	if err != nil {
		return nil, err
	}

	calendar, err := parseCalendar(data, config.ExcludeSummaries)
	if err != nil {
		return nil, fmt.Errorf("holiday_calendar: parse %s: %w", config.File, err)
	}

	return calendar, nil
}

func readCalendarFile(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("holiday_calendar: file path required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("holiday_calendar: read %s: %w", path, err)
	}

	return data, nil
}

func parseCalendar(data []byte, excludeSummaries []string) (*Calendar, error) {
	parsed, err := parseICS(data)
	if err != nil {
		return nil, err
	}

	calendar := &Calendar{
		days: make(map[string]struct{}),
	}

	if err := calendar.loadEvents(parsed.Events(), excludeSummaries); err != nil {
		return nil, err
	}

	return calendar, nil
}

func parseICS(data []byte) (*ics.Calendar, error) {
	if err := validateCalendarEnvelope(data); err != nil {
		return nil, err
	}

	return ics.ParseCalendar(bytes.NewReader(data))
}

func validateCalendarEnvelope(data []byte) error {
	data = bytes.TrimSpace(data)

	hasStart := bytes.HasPrefix(data, []byte("BEGIN:VCALENDAR"))
	hasEnd := bytes.HasSuffix(data, []byte("END:VCALENDAR"))

	if !hasEnd || !hasStart {
		return fmt.Errorf("expected a complete VCALENDAR")
	}

	return nil
}

func (calendar *Calendar) loadEvents(events []*ics.VEvent, excludeSummaries []string) error {
	for _, event := range events {
		if isExcludedEvent(event, excludeSummaries) {
			continue
		}

		start, end, err := eventDates(event)
		if err != nil {
			return fmt.Errorf("event %s: %w", event.Id(), err)
		}

		calendar.addDateRange(start, end)
	}

	return nil
}

func isExcludedEvent(event *ics.VEvent, excludeSummaries []string) bool {
	summary := event.GetProperty(ics.ComponentPropertySummary)

	if summary == nil {
		return false
	}

	title := ics.FromText(summary.Value)

	return slices.Contains(excludeSummaries, title)
}

func (calendar *Calendar) addDateRange(start, end time.Time) {
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		key := day.Format(time.DateOnly)
		calendar.days[key] = struct{}{}
	}
}

func eventDates(event *ics.VEvent) (time.Time, time.Time, error) {
	if err := validateEventProperties(event); err != nil {
		return time.Time{}, time.Time{}, err
	}

	start, err := readDate(event, ics.ComponentPropertyDtStart)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := readEventEnd(event, start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("DTEND must be after DTSTART")
	}

	return start, end, nil
}

func validateEventProperties(event *ics.VEvent) error {
	unsupported := []ics.ComponentProperty{
		ics.ComponentPropertyRrule,
		ics.ComponentPropertyRdate,
		ics.ComponentPropertyExdate,
		ics.ComponentPropertyExrule,
		ics.ComponentPropertyRecurrenceId,
		ics.ComponentPropertyDuration,
	}

	for _, property := range unsupported {
		if event.GetProperty(property) != nil {
			return fmt.Errorf("%s is unsupported; use explicit DTSTART/DTEND dates", property)
		}
	}

	return nil
}

func readEventEnd(event *ics.VEvent, start time.Time) (time.Time, error) {
	if event.GetProperty(ics.ComponentPropertyDtEnd) == nil {
		return start.AddDate(0, 0, 1), nil
	}

	return readDate(event, ics.ComponentPropertyDtEnd)
}

func readDate(event *ics.VEvent, name ics.ComponentProperty) (time.Time, error) {
	value, err := readDateValue(event, name)
	if err != nil {
		return time.Time{}, err
	}

	date, err := time.Parse("20060102", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s: %w", name, err)
	}

	return date, nil
}

func readDateValue(event *ics.VEvent, name ics.ComponentProperty) (string, error) {
	properties := event.GetProperties(name)

	if len(properties) != 1 || properties[0].GetValueType() != ics.ValueDataTypeDate {
		return "", fmt.Errorf("expected exactly one %s with VALUE=DATE", name)
	}

	property := properties[0]
	if _, hasTimezone := property.ICalParameters[string(ics.ParameterTzid)]; hasTimezone {
		return "", fmt.Errorf("%s with VALUE=DATE must not specify TZID", name)
	}

	return property.Value, nil
}

func scheduleLocation(tzOffset int64) *time.Location {
	return time.FixedZone("", int(-tzOffset*60))
}

func (calendar *Calendar) IsHoliday(timestamp time.Time, tzOffset int64) bool {
	if calendar == nil {
		return false
	}

	local := timestamp.In(scheduleLocation(tzOffset))
	key := local.Format(time.DateOnly)
	_, holiday := calendar.days[key]

	return holiday
}

func (calendar *Calendar) NextNonHoliday(timestamp time.Time, tzOffset int64) time.Time {
	for calendar.IsHoliday(timestamp, tzOffset) {
		timestamp = nextLocalMidnight(timestamp, tzOffset)
	}

	return timestamp.UTC()
}

func nextLocalMidnight(timestamp time.Time, tzOffset int64) time.Time {
	location := scheduleLocation(tzOffset)
	local := timestamp.In(location)
	year, month, day := local.Date()

	return time.Date(year, month, day+1, 0, 0, 0, 0, location)
}
