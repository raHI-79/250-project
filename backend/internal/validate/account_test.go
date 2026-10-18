package validate

import (
	"testing"
	"time"
)

func TestEmailInDomain(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		domain  string
		want    string
		wantErr string
	}{
		{name: "student address", value: "2023331079@student.sust.edu", domain: "student.sust.edu", want: "2023331079@student.sust.edu"},
		{name: "upper case is lowered", value: "Ratul@Student.SUST.edu", domain: "student.sust.edu", want: "ratul@student.sust.edu"},
		{name: "domain given in upper case", value: "ratul@student.sust.edu", domain: "STUDENT.SUST.EDU", want: "ratul@student.sust.edu"},
		{name: "spaces are trimmed", value: " teacher@sust.edu ", domain: "sust.edu", want: "teacher@sust.edu"},
		{
			name:    "teacher address on the student domain",
			value:   "teacher@sust.edu",
			domain:  "student.sust.edu",
			wantErr: "Email must end with @student.sust.edu.",
		},
		{
			name:    "student address on the teacher domain",
			value:   "2023331079@student.sust.edu",
			domain:  "sust.edu",
			wantErr: "Email must end with @sust.edu.",
		},
		{
			name:    "outside address",
			value:   "ratul@gmail.com",
			domain:  "student.sust.edu",
			wantErr: "Email must end with @student.sust.edu.",
		},
		{
			name:    "domain hidden in the local part",
			value:   "student.sust.edu@gmail.com",
			domain:  "student.sust.edu",
			wantErr: "Email must end with @student.sust.edu.",
		},
		{
			name:    "missing at sign",
			value:   "ratul.student.sust.edu",
			domain:  "student.sust.edu",
			wantErr: "Email must be a valid email address.",
		},
		{name: "empty", value: "", domain: "student.sust.edu", wantErr: "Email is required."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EmailInDomain("Email", tc.value, tc.domain)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestBirthDate(t *testing.T) {
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	const ageMessage = "Date of birth must belong to a person between 15 and 60 years old."
	const formatMessage = "Date of birth must be a date in the YYYY-MM-DD format."
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "student age", value: "2003-05-14", want: "2003-05-14"},
		{name: "exactly fifteen", value: "2011-10-18", want: "2011-10-18"},
		{name: "exactly sixty", value: "1966-10-18", want: "1966-10-18"},
		{name: "spaces are trimmed", value: " 1988-03-09 ", want: "1988-03-09"},
		{name: "one day short of fifteen", value: "2011-10-19", wantErr: ageMessage},
		{name: "fourteen", value: "2012-06-01", wantErr: ageMessage},
		{name: "just over sixty one", value: "1965-10-17", wantErr: ageMessage},
		{name: "in the future", value: "2030-01-01", wantErr: ageMessage},
		{name: "day first", value: "14-05-2003", wantErr: formatMessage},
		{name: "slashes", value: "2003/05/14", wantErr: formatMessage},
		{name: "not a date", value: "yesterday", wantErr: formatMessage},
		{name: "impossible day", value: "2003-02-30", wantErr: formatMessage},
		{name: "empty", value: "", wantErr: "Date of birth is required."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BirthDate("Date of birth", tc.value, now)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestPlainDate(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "joining date", value: "2015-01-15", want: "2015-01-15"},
		{name: "long past", value: "1979-09-01", want: "1979-09-01"},
		{name: "far future", value: "2099-12-31", want: "2099-12-31"},
		{name: "wrong order", value: "15-01-2015", wantErr: "Joining date must be a date in the YYYY-MM-DD format."},
		{name: "empty", value: "", wantErr: "Joining date is required."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PlainDate("Joining date", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}
