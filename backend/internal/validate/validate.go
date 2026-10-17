package validate

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxFieldLength    = 120
	MaxEmailLength    = 254
	MinPasswordLength = 8
)

var (
	emailPattern      = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s.]{2,}$`)
	phonePattern      = regexp.MustCompile(`^\+?[0-9]{10,15}$`)
	batchPattern      = regexp.MustCompile(`^[A-Z]{2,6}-[0-9]{2}$`)
	sectionPattern    = regexp.MustCompile(`^[A-Z]$`)
	courseCodePattern = regexp.MustCompile(`^[A-Z]{2,6}-?[0-9]{3}$`)
	teacherIDPattern  = regexp.MustCompile(`^[A-Za-z0-9-]{3,20}$`)
	bloodGroups       = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}
)

func Required(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required.", label)
	}
	if len(trimmed) > MaxFieldLength {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxFieldLength)
	}
	return trimmed, nil
}

func Email(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToLower(trimmed)
	if len(trimmed) > MaxEmailLength || !emailPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must be a valid email address.", label)
	}
	return trimmed, nil
}

func Phone(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	if !phonePattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must be 10 to 15 digits, with an optional leading +.", label)
	}
	return trimmed, nil
}

func PhoneDigits(value string) string {
	var digits strings.Builder
	for _, symbol := range value {
		if symbol >= '0' && symbol <= '9' {
			digits.WriteRune(symbol)
		}
	}
	cleaned := digits.String()
	if len(cleaned) > 11 && strings.HasPrefix(cleaned, "88") {
		cleaned = cleaned[2:]
	}
	return cleaned
}

func SamePhone(left, right string) bool {
	cleaned := PhoneDigits(left)
	return cleaned != "" && cleaned == PhoneDigits(right)
}

func Batch(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToUpper(trimmed)
	if !batchPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must look like CSE-23.", label)
	}
	return trimmed, nil
}

func Section(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToUpper(trimmed)
	if !sectionPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must be a single letter from A to Z.", label)
	}
	return trimmed, nil
}

func BloodGroup(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToUpper(strings.ReplaceAll(trimmed, " ", ""))
	for _, group := range bloodGroups {
		if trimmed == group {
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("%s must be one of %s.", label, strings.Join(bloodGroups, ", "))
}

func CourseCode(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToUpper(strings.ReplaceAll(trimmed, " ", ""))
	if !courseCodePattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must look like CSE-233 or CSE233.", label)
	}
	return trimmed, nil
}

func TeacherID(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	if !teacherIDPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s must be 3 to 20 letters, digits or hyphens.", label)
	}
	return trimmed, nil
}

func Password(label, password, confirm string) error {
	if password == "" {
		return fmt.Errorf("%s is required.", label)
	}
	if len([]rune(password)) < MinPasswordLength {
		return fmt.Errorf("%s must be at least %d characters long.", label, MinPasswordLength)
	}
	if len(password) > MaxFieldLength {
		return fmt.Errorf("%s must be at most %d characters long.", label, MaxFieldLength)
	}
	if password != confirm {
		return fmt.Errorf("%s and the confirmation do not match.", label)
	}
	return nil
}
