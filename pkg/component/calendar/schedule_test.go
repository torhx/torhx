package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/talav/civil"

	"github.com/torhx/torhx/pkg/component/calendar"
)

// mustHours returns the hours of a session of the given kind, with start and
// end written as wall-clock times such as "09:30".
func mustHours(t *testing.T, kind calendar.SessionKind, start, end string, daysBefore int) calendar.Hours {
	t.Helper()

	from, err := civil.NewTimeFromString(start)
	require.NoError(t, err)

	to, err := civil.NewTimeFromString(end)
	require.NoError(t, err)

	h, err := calendar.NewHours(kind, from, to, daysBefore)
	require.NoError(t, err)

	return h
}

func workweek() []time.Weekday {
	return []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
}

func everyDay() []time.Weekday {
	return []time.Weekday{
		time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday,
	}
}

// mustWeek returns the schedule that gives each of the weekdays the same
// hours, from the date from.
func mustWeek(t *testing.T, from string, weekdays []time.Weekday, hours ...calendar.Hours) calendar.WeeklySchedule {
	t.Helper()

	var days [7][]calendar.Hours
	for _, weekday := range weekdays {
		days[weekday] = hours
	}

	w, err := calendar.NewWeeklySchedule(mustDate(t, from), days)
	require.NoError(t, err)

	return w
}

func TestNewHoursRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		kind       calendar.SessionKind
		daysBefore int
		wantErr    string
	}{
		{name: "no kind", kind: "", wantErr: "unrecognised session kind"},
		{name: "negative days before", kind: calendar.SessionKindRegular, daysBefore: -1, wantErr: "days before must be 0 to 6"},
		{name: "a week before", kind: calendar.SessionKindRegular, daysBefore: 7, wantErr: "days before must be 0 to 6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := calendar.NewHours(tt.kind, civil.Time{}, civil.Time{}, tt.daysBefore)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewWeeklyScheduleRejects(t *testing.T) {
	t.Parallel()

	from := mustDate(t, "2026-01-01")
	morning := mustHours(t, calendar.SessionKindRegular, "09:00", "11:30", 0)
	afternoon := mustHours(t, calendar.SessionKindRegular, "12:30", "15:30", 0)
	overnight := mustHours(t, calendar.SessionKindRegular, "18:00", "17:00", 1)
	preMarket := mustHours(t, calendar.SessionKindExtended, "04:00", "09:00", 0)

	tests := []struct {
		name    string
		from    civil.Date
		monday  []calendar.Hours
		wantErr string
	}{
		{name: "no first date", from: civil.Date{}, monday: []calendar.Hours{morning}, wantErr: "first date is missing"},
		{name: "zero hours", from: from, monday: []calendar.Hours{{}}, wantErr: "are not hours"},
		{name: "out of order", from: from, monday: []calendar.Hours{afternoon, morning}, wantErr: "start before hours"},
		{name: "overnight session overlaps the morning", from: from, monday: []calendar.Hours{overnight, morning}, wantErr: "start before hours"},
		{name: "extended only", from: from, monday: []calendar.Hours{preMarket}, wantErr: "no regular session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var days [7][]calendar.Hours
			days[time.Monday] = tt.monday

			_, err := calendar.NewWeeklySchedule(tt.from, days)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
