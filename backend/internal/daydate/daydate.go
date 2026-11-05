package daydate

import (
	"fmt"
	"strings"
	"time"
)

const (
	Layout  = "2006-01-02"
	Display = "Mon 2 Jan 2006"
	Week    = 7
)

var dayNames = [Week]string{
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
}

func Zone(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}
	return loc
}

func Of(at time.Time, loc *time.Location) string {
	return at.In(Zone(loc)).Format(Layout)
}

func Today(now time.Time, loc *time.Location) string {
	return Of(now, loc)
}

func Tomorrow(now time.Time, loc *time.Location) string {
	return Shift(Of(now, loc), 1)
}

func Yesterday(now time.Time, loc *time.Location) string {
	return Shift(Of(now, loc), -1)
}

func Parse(value string, loc *time.Location) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("daydate: the date is empty")
	}
	parsed, err := time.ParseInLocation(Layout, trimmed, Zone(loc))
	if err != nil {
		return time.Time{}, fmt.Errorf("daydate: %q is not a date like 2026-11-06", trimmed)
	}
	return parsed, nil
}

func Valid(value string) bool {
	_, err := Parse(value, time.UTC)
	return err == nil
}

func Start(date string, loc *time.Location) (time.Time, error) {
	parsed, err := Parse(date, loc)
	if err != nil {
		return time.Time{}, err
	}
	zone := Zone(loc)
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, zone), nil
}

func End(date string, loc *time.Location) (time.Time, error) {
	start, err := Start(date, loc)
	if err != nil {
		return time.Time{}, err
	}
	return start.AddDate(0, 0, 1), nil
}

func StartOf(now time.Time, loc *time.Location) time.Time {
	zone := Zone(loc)
	here := now.In(zone)
	return time.Date(here.Year(), here.Month(), here.Day(), 0, 0, 0, 0, zone)
}

func Shift(date string, days int) string {
	parsed, err := Parse(date, time.UTC)
	if err != nil {
		return date
	}
	return parsed.AddDate(0, 0, days).Format(Layout)
}

func Between(from, to string) int {
	start, err := Parse(from, time.UTC)
	if err != nil {
		return 0
	}
	end, err := Parse(to, time.UTC)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Hours() / 24)
}

func Weekday(date string) int {
	parsed, err := Parse(date, time.UTC)
	if err != nil {
		return -1
	}
	return int(parsed.Weekday())
}

func DayName(day int) string {
	if day < 0 || day >= Week {
		return ""
	}
	return dayNames[day]
}

func WeekdayName(date string) string {
	return DayName(Weekday(date))
}

func WeekdayNumber(name string) (int, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	for index, day := range dayNames {
		if strings.ToLower(day) == trimmed || strings.ToLower(day[:3]) == trimmed {
			return index, true
		}
	}
	return 0, false
}

func WeekNumber(date string) (int, int) {
	parsed, err := Parse(date, time.UTC)
	if err != nil {
		return 0, 0
	}
	return parsed.ISOWeek()
}

func Readable(date string) string {
	parsed, err := Parse(date, time.UTC)
	if err != nil {
		return date
	}
	return parsed.Format(Display)
}

func Clock(at time.Time, loc *time.Location) string {
	return at.In(Zone(loc)).Format("15:04")
}

func Minutes(clock string) (int, bool) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(clock))
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func Range(from, to string) []string {
	days := Between(from, to)
	if days < 0 {
		return nil
	}
	dates := make([]string, 0, days+1)
	for day := 0; day <= days; day++ {
		dates = append(dates, Shift(from, day))
	}
	return dates
}
