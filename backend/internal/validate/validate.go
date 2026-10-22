package validate

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

const (
	MaxFieldLength    = 120
	MaxEmailLength    = 254
	MinPasswordLength = 8
	MinCourseCode     = 3
	MaxCourseCode     = 25
	MinCredit         = 0.5
	MaxCredit         = 6.0
	CreditStep        = 0.25
	MinUserIDLength   = 4
	MaxUserIDLength   = 20
	MinAge            = 15
	MaxAge            = 60
	MessagePhone      = "Phone number must be 11 digits, for example 01712345678."
	MessageCourseCode = "Course code may only use letters, digits, spaces and hyphens, for example CSE 2243 7637D."

	MessageUserIDRequired   = "User ID is required."
	MessageUserIDLength     = "User ID must be 4 to 20 characters long."
	MessageUserIDCharacters = "User ID may only use letters, digits, dots, underscores and hyphens."
	MessageUserIDDigitsOnly = "User ID cannot be digits only."
	MessageUserIDStart      = "User ID must start with a letter."
	MessageUserIDReserved   = "This User ID is reserved."
	SectionA                = "A"
	SectionB                = "B"
	LabelBothSections       = "Both sections"
)

var (
	emailPattern      = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s.]{2,}$`)
	phonePattern      = regexp.MustCompile(`^01[3-9][0-9]{8}$`)
	batchPattern      = regexp.MustCompile(`^[A-Z]{2,6}-[0-9]{2}$`)
	sectionPattern    = regexp.MustCompile(`^[AB]$`)
	courseCodePattern = regexp.MustCompile(`^[A-Z0-9 ]+$`)
	courseCodeDigit   = regexp.MustCompile(`[0-9]`)
	userIDPattern     = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	userIDDigits      = regexp.MustCompile(`^[0-9]+$`)
	userIDStart       = regexp.MustCompile(`^[A-Za-z]`)
	bloodGroups       = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}
	reservedUserIDs   = []string{"admin", "office", "root", "support", "system", "teacher", "student"}
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
		return "", errors.New(MessagePhone)
	}
	return trimmed, nil
}

func EmailInDomain(label, value, domain string) (string, error) {
	trimmed, err := Email(label, value)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(trimmed, "@"+strings.ToLower(domain)) {
		return "", fmt.Errorf("%s must end with @%s.", label, strings.ToLower(domain))
	}
	return trimmed, nil
}

func BirthDate(label, value string, now time.Time) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	parsed, err := time.ParseInLocation(DateLayout, trimmed, now.Location())
	if err != nil {
		return "", fmt.Errorf("%s must be a date in the YYYY-MM-DD format.", label)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	oldest := today.AddDate(-MaxAge-1, 0, 1)
	youngest := today.AddDate(-MinAge, 0, 0)
	if parsed.Before(oldest) || parsed.After(youngest) {
		return "", fmt.Errorf("%s must belong to a person between %d and %d years old.", label, MinAge, MaxAge)
	}
	return parsed.Format(DateLayout), nil
}

func PlainDate(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	parsed, err := time.Parse(DateLayout, trimmed)
	if err != nil {
		return "", fmt.Errorf("%s must be a date in the YYYY-MM-DD format.", label)
	}
	return parsed.Format(DateLayout), nil
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
		return "", fmt.Errorf("%s must be A or B.", label)
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

func NormalizeCourseCode(value string) string {
	upper := strings.ToUpper(strings.ReplaceAll(value, "-", " "))
	return strings.Join(strings.Fields(upper), " ")
}

func SqueezeCourseCode(value string) string {
	return strings.ReplaceAll(NormalizeCourseCode(value), " ", "")
}

func CourseCode(label, value string) (string, error) {
	normalized := NormalizeCourseCode(value)
	if normalized == "" {
		return "", fmt.Errorf("%s is required.", label)
	}
	length := len([]rune(normalized))
	if length < MinCourseCode || length > MaxCourseCode {
		return "", fmt.Errorf("%s must be %d to %d characters long.", label, MinCourseCode, MaxCourseCode)
	}
	if !courseCodePattern.MatchString(normalized) {
		return "", errors.New(MessageCourseCode)
	}
	if !courseCodeDigit.MatchString(normalized) {
		return "", fmt.Errorf("%s must contain at least one digit.", label)
	}
	return normalized, nil
}

func Credit(label string, value float64) (float64, error) {
	if value == 0 {
		return 0, fmt.Errorf("%s is required.", label)
	}
	steps := math.Round(value / CreditStep)
	if math.Abs(value-steps*CreditStep) > 1e-9 {
		return 0, fmt.Errorf("%s must move in steps of %.2f.", label, CreditStep)
	}
	rounded := steps * CreditStep
	if rounded < MinCredit || rounded > MaxCredit {
		return 0, fmt.Errorf("%s must be between %.1f and %.1f.", label, MinCredit, MaxCredit)
	}
	return rounded, nil
}

func UserID(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New(MessageUserIDRequired)
	}
	length := len([]rune(trimmed))
	if length < MinUserIDLength || length > MaxUserIDLength {
		return "", errors.New(MessageUserIDLength)
	}
	if !userIDPattern.MatchString(trimmed) {
		return "", errors.New(MessageUserIDCharacters)
	}
	if userIDDigits.MatchString(trimmed) {
		return "", errors.New(MessageUserIDDigitsOnly)
	}
	if !userIDStart.MatchString(trimmed) {
		return "", errors.New(MessageUserIDStart)
	}
	lowered := strings.ToLower(trimmed)
	for _, reserved := range reservedUserIDs {
		if lowered == reserved {
			return "", errors.New(MessageUserIDReserved)
		}
	}
	return trimmed, nil
}

func ReservedUserIDs() []string {
	return append([]string(nil), reservedUserIDs...)
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
