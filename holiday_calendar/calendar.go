package holiday_calendar

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"time"

	ics "github.com/arran4/golang-ical"
)

// Calendar contains holidays as set.
type Calendar struct {
	days map[string]struct{}
}

// New creates a holiday calendar.
func New(file string, excludeSummaries []string) (*Calendar, error) {
	data, err := readCalendarFile(file)
	if err != nil {
		return nil, err
	}

	calendar, err := parseCalendar(data, excludeSummaries)
	if err != nil {
		return nil, fmt.Errorf("holiday_calendar: parse %s: %w", file, err)
	}

	return calendar, nil
}

// readCalendarFile reads calendar file via forwarded path.
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

// parseCalendar parses raw ics calendar data into a Calendar, skipping excludeSummaries slice.
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

// parseICS parses ics calendar data via third-party parser with validation.
func parseICS(data []byte) (*ics.Calendar, error) {
	if err := validateCalendarEnvelope(data); err != nil {
		return nil, err
	}

	return ics.ParseCalendar(bytes.NewReader(data))
}

// validateCalendarEnvelope checks if ics is valid and complete.
func validateCalendarEnvelope(data []byte) error {
	data = bytes.TrimSpace(data)

	hasStart := bytes.HasPrefix(data, []byte("BEGIN:VCALENDAR"))
	hasEnd := bytes.HasSuffix(data, []byte("END:VCALENDAR"))

	if !hasEnd || !hasStart {
		return fmt.Errorf("expected a complete VCALENDAR")
	}

	return nil
}

// loadEvents adds every non-excluded event to the calendar.
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

// isExcludedEvent excludes events with certain SUMMARY.
func isExcludedEvent(event *ics.VEvent, excludeSummaries []string) bool {
	summary := event.GetProperty(ics.ComponentPropertySummary)

	if summary == nil {
		return false
	}

	title := ics.FromText(summary.Value)

	return slices.Contains(excludeSummaries, title)
}

// addDateRange adds every day in half-open interval [start, end) to rhe calendar.
func (calendar *Calendar) addDateRange(start, end time.Time) {
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		key := day.Format(time.DateOnly)
		calendar.days[key] = struct{}{}
	}
}

// eventDates returns the start and end dates of event.
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

// validateEventProperties rejects properties that imply recurrence or duration, expects only explicit DTSTART, DTEND events.
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

// readEventEnd returns DTEND or start+1 day if DTEND absent.
func readEventEnd(event *ics.VEvent, start time.Time) (time.Time, error) {
	if event.GetProperty(ics.ComponentPropertyDtEnd) == nil {
		return start.AddDate(0, 0, 1), nil
	}

	return readDate(event, ics.ComponentPropertyDtEnd)
}

// readDate parses an all-day date property into a UTC time. only VALUE=DATE without TZID are accepted.
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

// readDateValue returns raw value of single VALUE=DATE property named name. rejects multiple occurrences and TZID parametr.
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

// scheduleLocation returns a fixed timezone for given schedule offset. converts tzOffset in minutes and the converts the sign, cause in schedule it is stored with "-" for eastern regions and "+" for western.
func scheduleLocation(tzOffset int64) *time.Location {
	return time.FixedZone("", int(-tzOffset*60))
}

// IsHoliday return true if timestamp falls on a holiday in calendar, using tzOffset to resolve to the local calendar date.
func (calendar *Calendar) IsHoliday(timestamp time.Time, tzOffset int64) bool {
	if calendar == nil {
		return false
	}

	local := timestamp.In(scheduleLocation(tzOffset))
	key := local.Format(time.DateOnly)
	_, holiday := calendar.days[key]

	return holiday
}

// NextNonHoliday returns the earliest moment at or after timestamp that does not fall an a holiday.
func (calendar *Calendar) NextNonHoliday(timestamp time.Time, tzOffset int64) time.Time {
	for calendar.IsHoliday(timestamp, tzOffset) {
		timestamp = nextLocalMidnight(timestamp, tzOffset)
	}

	return timestamp.UTC()
}

// nextLocalMidnight returns the start of the next local day relative to timestamp in the zone difined by tzOffset.
func nextLocalMidnight(timestamp time.Time, tzOffset int64) time.Time {
	location := scheduleLocation(tzOffset)
	local := timestamp.In(location)
	year, month, day := local.Date()

	return time.Date(year, month, day+1, 0, 0, 0, 0, location)
}
