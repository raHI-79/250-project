package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	SectionAll   = "ALL"
	RoutineClass = "class"
	RoutineLab   = "lab"
)

var (
	clockPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):([0-5][0-9])$`)
	dayNames     = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
)

func RoutineSection(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToUpper(trimmed)
	if trimmed == SectionAll || sectionPattern.MatchString(trimmed) {
		return trimmed, nil
	}
	return "", fmt.Errorf("%s must be A, B or ALL.", label)
}

func Sections() []string {
	return []string{SectionA, SectionB}
}

func SectionsWithAll() []string {
	return []string{SectionA, SectionB, SectionAll}
}

func ClockTime(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	if !clockPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must look like 09:30.", label)
	}
	return trimmed, nil
}

func Minutes(clock string) int {
	parts := strings.SplitN(clock, ":", 2)
	if len(parts) != 2 {
		return -1
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return -1
	}
	return hours*60 + minutes
}

func Clock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

func DayOfWeek(label string, value int) (int, error) {
	if value < 0 || value >= len(dayNames) {
		return 0, fmt.Errorf("%s must be a number from 0 (Sunday) to 6 (Saturday).", label)
	}
	return value, nil
}

func DayName(day int) string {
	if day < 0 || day >= len(dayNames) {
		return ""
	}
	return dayNames[day]
}

func RoutineType(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToLower(trimmed)
	if trimmed != RoutineClass && trimmed != RoutineLab {
		return "", fmt.Errorf("%s must be %s or %s.", label, RoutineClass, RoutineLab)
	}
	return trimmed, nil
}
