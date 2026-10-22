package validate

import (
	"fmt"
	"strings"
)

const DefaultSemester = "1-1"

var (
	semesters      = []string{"1-1", "1-2", "2-1", "2-2", "3-1", "3-2", "4-1", "4-2"}
	semesterYears  = []string{"1st year", "2nd year", "3rd year", "4th year"}
	semesterHalves = []string{"1st semester", "2nd semester"}
)

func Semesters() []string {
	return append([]string(nil), semesters...)
}

func Semester(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	for _, semester := range semesters {
		if trimmed == semester {
			return semester, nil
		}
	}
	return "", fmt.Errorf("%s must be one of %s.", label, strings.Join(semesters, ", "))
}

func SemesterLabel(value string) string {
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return value
	}
	year, half := parts[0][0]-'1', parts[1][0]-'1'
	if int(year) >= len(semesterYears) || int(half) >= len(semesterHalves) {
		return value
	}
	return semesterYears[year] + " " + semesterHalves[half]
}
