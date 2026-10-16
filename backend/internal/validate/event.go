package validate

import (
	"fmt"
	"strings"
	"time"
)

const (
	DateLayout = "2006-01-02"
	EventClass = "class"
	EventExam  = "exam"
	EventExtra = "extra"
)

var eventTypes = []string{EventClass, EventExam, EventExtra}

func EventType(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToLower(trimmed)
	for _, eventType := range eventTypes {
		if trimmed == eventType {
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("%s must be one of %s.", label, strings.Join(eventTypes, ", "))
}

func Date(label, value string, loc *time.Location) (time.Time, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.ParseInLocation(DateLayout, trimmed, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must look like 2026-10-17.", label)
	}
	return parsed, nil
}

func FutureDate(label, value string, now time.Time, loc *time.Location) (time.Time, error) {
	parsed, err := Date(label, value, loc)
	if err != nil {
		return time.Time{}, err
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if !parsed.After(today) {
		return time.Time{}, fmt.Errorf("%s must be tomorrow or later.", label)
	}
	return parsed, nil
}

func DateString(at time.Time) string {
	return at.Format(DateLayout)
}
