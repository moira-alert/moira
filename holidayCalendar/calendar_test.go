package holidayCalendar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCalendarDatesAndOffsets(t *testing.T) {
	calendar, err := New(Config{Enabled: true, File: "testdata/holidays.ics"})
	require.NoError(t, err)

	for _, test := range []struct {
		name     string
		now      string
		offset   int64
		holiday  bool
		wantNext string
	}{
		{"before local holiday", "2026-04-30T20:59:59Z", -180, false, "2026-04-30T20:59:59Z"},
		{"local holiday starts", "2026-04-30T21:00:00Z", -180, true, "2026-05-01T21:00:00Z"},
		{"local holiday ends", "2026-05-01T21:00:00Z", -180, false, "2026-05-01T21:00:00Z"},
		{"same instant is not a holiday in UTC", "2026-04-30T21:00:00Z", 0, false, "2026-04-30T21:00:00Z"},
		{"UTC holiday", "2026-05-01T12:00:00Z", 0, true, "2026-05-02T00:00:00Z"},
		{"positive offset before holiday", "2026-05-01T04:59:59Z", 300, false, "2026-05-01T04:59:59Z"},
		{"positive offset starts holiday", "2026-05-01T05:00:00Z", 300, true, "2026-05-02T05:00:00Z"},
		{"half hour offset", "2026-04-30T18:30:00Z", -330, true, "2026-05-01T18:30:00Z"},
		{"single day without DTEND", "2026-01-09T12:00:00Z", -180, true, "2026-01-09T21:00:00Z"},
		{"consecutive events across year boundary", "2025-12-31T12:00:00Z", -180, true, "2026-01-09T21:00:00Z"},
		{"ordinary Saturday", "2026-01-10T12:00:00Z", -180, false, "2026-01-10T12:00:00Z"},
		{"ordinary Sunday", "2026-01-11T12:00:00Z", -180, false, "2026-01-11T12:00:00Z"},
		{"holiday on Saturday", "2026-05-09T12:00:00Z", -180, true, "2026-05-09T21:00:00Z"},
		{"date absent from ICS", "2026-02-23T12:00:00Z", -180, false, "2026-02-23T12:00:00Z"},
		{"no extrapolation to another year", "2027-01-09T12:00:00Z", -180, false, "2027-01-09T12:00:00Z"},
		{"offset stays fixed across DST date", "2026-03-08T05:00:00Z", 300, true, "2026-03-10T05:00:00Z"},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := parseCalendarTime(t, test.now)
			wantNext := parseCalendarTime(t, test.wantNext)

			require.Equal(t, test.holiday, calendar.IsHoliday(now, test.offset))
			next := calendar.NextNonHoliday(now, test.offset)
			require.Equal(t, wantNext, next)
			require.Same(t, time.UTC, next.Location())
			require.False(t, calendar.IsHoliday(next, test.offset))
		})
	}
}

func TestCalendarUsesScheduleOffsetInsteadOfTimestampLocation(t *testing.T) {
	calendar, err := New(Config{Enabled: true, File: "testdata/holidays.ics"})
	require.NoError(t, err)

	now := parseCalendarTime(t, "2026-04-30T21:00:00Z").In(time.FixedZone("input", -7*60*60))
	require.True(t, calendar.IsHoliday(now, -180))
	require.False(t, calendar.IsHoliday(now, 0))
	require.Equal(t, now.UTC(), calendar.NextNonHoliday(now, 0))
	require.Equal(t, parseCalendarTime(t, "2026-05-01T21:00:00Z"), calendar.NextNonHoliday(now, -180))
	// Reusing the calendar for another schedule must not change its behavior.
	require.True(t, calendar.IsHoliday(now, -180))
}

func TestCalendarConfig(t *testing.T) {
	missingFile := filepath.Join(t.TempDir(), "missing.ics")

	for _, test := range []struct {
		name   string
		config Config
		err    string
	}{
		{"disabled", Config{}, ""},
		{"disabled does not read file", Config{File: missingFile}, ""},
		{"enabled without timezone", Config{Enabled: true, File: "testdata/holidays.ics"}, ""},
		{"missing file setting", Config{Enabled: true}, "file path required"},
		{"unreadable file", Config{Enabled: true, File: missingFile}, "read"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calendar, err := New(test.config)
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				require.Nil(t, calendar)

				return
			}

			require.NoError(t, err)
			if test.config.Enabled {
				require.NotNil(t, calendar)
			} else {
				require.Nil(t, calendar)
				now := parseCalendarTime(t, "2026-01-09T12:00:00Z")
				require.False(t, calendar.IsHoliday(now, -180))
				require.Equal(t, now, calendar.NextNonHoliday(now, -180))
			}
		})
	}
}

func TestCalendarLineEndings(t *testing.T) {
	for _, separator := range []string{"\n", "\r\n"} {
		t.Run(fmt.Sprintf("%q", separator), func(t *testing.T) {
			contents := strings.ReplaceAll(calendarDocument("DTSTART;VALUE=DATE:20260109"), "\n", separator)
			path := writeCalendarFile(t, contents)
			calendar, err := New(Config{Enabled: true, File: path})
			require.NoError(t, err)
			require.True(t, calendar.IsHoliday(parseCalendarTime(t, "2026-01-09T12:00:00Z"), 0))
		})
	}
}

func TestCalendarRejectsInvalidDocuments(t *testing.T) {
	for _, test := range []struct {
		name     string
		contents string
		err      string
	}{
		{"empty", "", "complete VCALENDAR"},
		{"missing beginning", "VERSION:2.0\nEND:VCALENDAR\n", "complete VCALENDAR"},
		{"truncated", "BEGIN:VCALENDAR\nVERSION:2.0\n", "complete VCALENDAR"},
		{"invalid property", "BEGIN:VCALENDAR\ninvalid\nEND:VCALENDAR\n", "parse"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeCalendarFile(t, test.contents)
			calendar, err := New(Config{Enabled: true, File: path})
			require.ErrorContains(t, err, test.err)
			require.ErrorContains(t, err, path)
			require.Nil(t, calendar)
		})
	}
}

func TestCalendarRejectsInvalidEventDates(t *testing.T) {
	for _, test := range []struct {
		name       string
		properties string
		err        string
	}{
		{"missing start", "SUMMARY:Missing start", "DTSTART"},
		{"timed start", "DTSTART:20260109T000000Z", "VALUE=DATE"},
		{"missing DATE type", "DTSTART:20260109", "VALUE=DATE"},
		{"duplicate start", "DTSTART;VALUE=DATE:20260109\nDTSTART;VALUE=DATE:20260110", "exactly one DTSTART"},
		{"timezone on start", "DTSTART;VALUE=DATE;TZID=Europe/Moscow:20260109", "TZID"},
		{"invalid start date", "DTSTART;VALUE=DATE:20260230", "invalid DTSTART"},
		{"invalid end date", "DTSTART;VALUE=DATE:20260109\nDTEND;VALUE=DATE:20260230", "invalid DTEND"},
		{"timed end", "DTSTART;VALUE=DATE:20260109\nDTEND:20260110T000000Z", "VALUE=DATE"},
		{"duplicate end", "DTSTART;VALUE=DATE:20260109\nDTEND;VALUE=DATE:20260110\nDTEND;VALUE=DATE:20260111", "exactly one DTEND"},
		{"timezone on end", "DTSTART;VALUE=DATE:20260109\nDTEND;VALUE=DATE;TZID=Europe/Moscow:20260110", "TZID"},
		{"empty range", "DTSTART;VALUE=DATE:20260109\nDTEND;VALUE=DATE:20260109", "DTEND must be after DTSTART"},
		{"reversed range", "DTSTART;VALUE=DATE:20260109\nDTEND;VALUE=DATE:20260108", "DTEND must be after DTSTART"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeCalendarFile(t, calendarDocument(test.properties))
			calendar, err := New(Config{Enabled: true, File: path})
			require.ErrorContains(t, err, test.err)
			require.ErrorContains(t, err, "event test-0")
			require.Nil(t, calendar)
		})
	}
}

func TestCalendarRejectsUnsupportedEventProperties(t *testing.T) {
	for _, property := range []string{
		"RRULE:FREQ=YEARLY",
		"RDATE;VALUE=DATE:20270109",
		"EXDATE;VALUE=DATE:20260109",
		"EXRULE:FREQ=YEARLY",
		"RECURRENCE-ID;VALUE=DATE:20260109",
		"DURATION:P1D",
	} {
		t.Run(property, func(t *testing.T) {
			path := writeCalendarFile(t, calendarDocument("DTSTART;VALUE=DATE:20260109\n"+property))
			calendar, err := New(Config{Enabled: true, File: path})
			require.ErrorContains(t, err, "unsupported")
			require.Nil(t, calendar)
		})
	}
}

func TestCalendarExcludedSummaries(t *testing.T) {
	path := writeCalendarFile(t, calendarDocument(
		"DTSTART;VALUE=DATE:20260102\nSUMMARY:Shortened\\, day",
		"DTSTART;VALUE=DATE:20260103\nSUMMARY:shortened\\, day",
		"DTSTART;VALUE=DATE:20260104\nSUMMARY:Shortened\\, day",
		"DTSTART;VALUE=DATE:20260104\nSUMMARY:Holiday",
		"DTSTART;VALUE=DATE:20260105",
		"DTSTART;VALUE=DATE:20260106\nSUMMARY:* Shortened day",
	))
	calendar, err := New(Config{
		Enabled:          true,
		File:             path,
		ExcludeSummaries: []string{"Shortened, day", "* Shortened day"},
	})
	require.NoError(t, err)

	for day, want := range map[int]bool{2: false, 3: true, 4: true, 5: true, 6: false} {
		require.Equal(t, want, calendar.IsHoliday(time.Date(2026, time.January, day, 12, 0, 0, 0, time.UTC), 0), "day %d", day)
	}

	unfiltered, err := New(Config{Enabled: true, File: path})
	require.NoError(t, err)
	require.True(t, unfiltered.IsHoliday(parseCalendarTime(t, "2026-01-02T12:00:00Z"), 0))
}

func TestCalendarLoadsSnapshotOfFile(t *testing.T) {
	path := writeCalendarFile(t, calendarDocument("DTSTART;VALUE=DATE:20260109"))
	config := Config{Enabled: true, File: path}
	calendar, err := New(config)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path, []byte(calendarDocument("DTSTART;VALUE=DATE:20260112")), 0o600))
	reloaded, err := New(config)
	require.NoError(t, err)

	january9 := parseCalendarTime(t, "2026-01-09T12:00:00Z")
	january12 := parseCalendarTime(t, "2026-01-12T12:00:00Z")
	require.True(t, calendar.IsHoliday(january9, 0))
	require.False(t, calendar.IsHoliday(january12, 0))
	require.False(t, reloaded.IsHoliday(january9, 0))
	require.True(t, reloaded.IsHoliday(january12, 0))
}

func TestEmptyCalendarAllowsEveryDate(t *testing.T) {
	path := writeCalendarFile(t, calendarDocument())
	calendar, err := New(Config{Enabled: true, File: path})
	require.NoError(t, err)

	now := parseCalendarTime(t, "2026-01-09T12:00:00Z")
	require.False(t, calendar.IsHoliday(now, -180))
	require.Equal(t, now, calendar.NextNonHoliday(now, -180))
}

func writeCalendarFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "holidays.ics")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func calendarDocument(events ...string) string {
	var contents strings.Builder

	contents.WriteString("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Moira//Tests//EN\n")

	for index, properties := range events {
		fmt.Fprintf(&contents, "BEGIN:VEVENT\nUID:test-%d\nDTSTAMP:20260101T000000Z\n%s\nEND:VEVENT\n", index, properties)
	}

	contents.WriteString("END:VCALENDAR\n")

	return contents.String()
}

func parseCalendarTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return parsed
}
