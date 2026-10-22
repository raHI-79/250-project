package validate

import (
	"strings"
	"testing"
)

func TestRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "plain value", value: "Ratul Islam", want: "Ratul Islam"},
		{name: "surrounding spaces are trimmed", value: "  Ratul Islam  ", want: "Ratul Islam"},
		{name: "empty", value: "", wantErr: "Full name is required."},
		{name: "only spaces", value: "   ", wantErr: "Full name is required."},
		{name: "only a tab", value: "\t", wantErr: "Full name is required."},
		{
			name:    "too long",
			value:   strings.Repeat("x", MaxFieldLength+1),
			wantErr: "Full name must be at most 120 characters long.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Required("Full name", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "simple address", value: "student@sust.edu", want: "student@sust.edu"},
		{name: "uppercase is lowered", value: "Student@SUST.edu", want: "student@sust.edu"},
		{name: "spaces are trimmed", value: " student@sust.edu ", want: "student@sust.edu"},
		{name: "subdomain", value: "a.b@mail.sust.edu", want: "a.b@mail.sust.edu"},
		{name: "plus tag", value: "student+cse@sust.edu", want: "student+cse@sust.edu"},
		{name: "empty", value: "", wantErr: "Email is required."},
		{name: "no at sign", value: "student.sust.edu", wantErr: "Email must be a valid email address."},
		{name: "two at signs", value: "a@b@sust.edu", wantErr: "Email must be a valid email address."},
		{name: "no domain dot", value: "student@sust", wantErr: "Email must be a valid email address."},
		{name: "no local part", value: "@sust.edu", wantErr: "Email must be a valid email address."},
		{name: "inner space", value: "stu dent@sust.edu", wantErr: "Email must be a valid email address."},
		{name: "one letter tld", value: "student@sust.e", wantErr: "Email must be a valid email address."},
		{name: "trailing dot", value: "student@sust.edu.", wantErr: "Email must be a valid email address."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Email("Email", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestPhone(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "grameenphone", value: "01712345678", want: "01712345678"},
		{name: "robi", value: "01812345678", want: "01812345678"},
		{name: "banglalink", value: "01912345678", want: "01912345678"},
		{name: "teletalk", value: "01512345678", want: "01512345678"},
		{name: "airtel", value: "01612345678", want: "01612345678"},
		{name: "lowest allowed operator digit", value: "01312345678", want: "01312345678"},
		{name: "spaces are trimmed", value: " 01712345678 ", want: "01712345678"},
		{name: "empty", value: "", wantErr: "Phone is required."},
		{name: "ten digits", value: "1712345678", wantErr: MessagePhone},
		{name: "twelve digits", value: "017123456789", wantErr: MessagePhone},
		{name: "country code", value: "+8801712345678", wantErr: MessagePhone},
		{name: "country code without a plus", value: "8801712345678", wantErr: MessagePhone},
		{name: "operator digit zero", value: "01012345678", wantErr: MessagePhone},
		{name: "operator digit two", value: "01212345678", wantErr: MessagePhone},
		{name: "does not start with zero one", value: "02712345678", wantErr: MessagePhone},
		{name: "nine digits", value: "017123456", wantErr: MessagePhone},
		{name: "letters", value: "01712345abc", wantErr: MessagePhone},
		{name: "inner dash", value: "01712-345678", wantErr: MessagePhone},
		{name: "inner space", value: "017 1234 5678", wantErr: MessagePhone},
		{name: "plus in the middle", value: "0171+2345678", wantErr: MessagePhone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Phone("Phone", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestBatch(t *testing.T) {
	const message = "Batch must look like CSE-23."
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "standard batch", value: "CSE-23", want: "CSE-23"},
		{name: "lowercase is raised", value: "cse-23", want: "CSE-23"},
		{name: "spaces are trimmed", value: " CSE-23 ", want: "CSE-23"},
		{name: "two letter department", value: "EE-21", want: "EE-21"},
		{name: "empty", value: "", wantErr: "Batch is required."},
		{name: "missing dash", value: "CSE23", wantErr: message},
		{name: "four digit year", value: "CSE-2023", wantErr: message},
		{name: "one digit year", value: "CSE-2", wantErr: message},
		{name: "digits only", value: "23", wantErr: message},
		{name: "inner space", value: "CSE 23", wantErr: message},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Batch("Batch", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestSection(t *testing.T) {
	const message = "Section must be A or B."
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "section A", value: "A", want: "A"},
		{name: "lowercase is raised", value: "b", want: "B"},
		{name: "spaces are trimmed", value: " A ", want: "A"},
		{name: "empty", value: "", wantErr: "Section is required."},
		{name: "a third section", value: "C", wantErr: message},
		{name: "last letter", value: "Z", wantErr: message},
		{name: "two letters", value: "AB", wantErr: message},
		{name: "digit", value: "1", wantErr: message},
		{name: "symbol", value: "-", wantErr: message},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Section("Section", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestBloodGroup(t *testing.T) {
	const message = "Blood group must be one of A+, A-, B+, B-, AB+, AB-, O+, O-."
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "o positive", value: "O+", want: "O+"},
		{name: "ab negative", value: "AB-", want: "AB-"},
		{name: "lowercase is raised", value: "ab+", want: "AB+"},
		{name: "inner space is removed", value: "O +", want: "O+"},
		{name: "empty", value: "", wantErr: "Blood group is required."},
		{name: "unknown group", value: "C+", wantErr: message},
		{name: "missing sign", value: "O", wantErr: message},
		{name: "word", value: "positive", wantErr: message},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BloodGroup("Blood group", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestCourseCode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{name: "a hyphen becomes a space", value: "CSE-233", want: "CSE 233"},
		{name: "without a separator", value: "CSE233", want: "CSE233"},
		{name: "lowercase is raised", value: "cse 2243 7637d", want: "CSE 2243 7637D"},
		{name: "repeated spaces collapse", value: "CSE   2243    7637D", want: "CSE 2243 7637D"},
		{name: "outer spaces are trimmed", value: "  cse 1111  ", want: "CSE 1111"},
		{name: "a missing letter prefix is fine", value: "2243 7637D", want: "2243 7637D"},
		{name: "hyphens and spaces mix", value: "CSE-2243 7637-D", want: "CSE 2243 7637 D"},
		{name: "empty", value: "", wantErr: "Course code is required."},
		{name: "only spaces", value: "   ", wantErr: "Course code is required."},
		{name: "too short", value: "A1", wantErr: "Course code must be 3 to 25 characters long."},
		{
			name:    "too long",
			value:   "CSE 2243 7637D 1234567890 12",
			wantErr: "Course code must be 3 to 25 characters long.",
		},
		{name: "no digit", value: "CSE CODE", wantErr: "Course code must contain at least one digit."},
		{name: "an underscore", value: "CSE_2243", wantErr: MessageCourseCode},
		{name: "a slash", value: "CSE/2243", wantErr: MessageCourseCode},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CourseCode("Course code", tc.value)
			checkResult(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestSqueezeCourseCode(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "spaces go", value: "CSE 2243 7637D", want: "CSE22437637D"},
		{name: "hyphens go", value: "cse-233", want: "CSE233"},
		{name: "already squeezed", value: "CSE233", want: "CSE233"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SqueezeCourseCode(tc.value); got != tc.want {
				t.Errorf("SqueezeCourseCode(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		confirm  string
		wantErr  string
	}{
		{name: "matching and long enough", password: "sust-250", confirm: "sust-250"},
		{name: "exactly eight characters", password: "12345678", confirm: "12345678"},
		{name: "spaces are allowed inside", password: "two words", confirm: "two words"},
		{name: "empty", password: "", confirm: "", wantErr: "Password is required."},
		{
			name:     "too short",
			password: "sust-25",
			confirm:  "sust-25",
			wantErr:  "Password must be at least 8 characters long.",
		},
		{
			name:     "confirmation does not match",
			password: "sust-250",
			confirm:  "sust-251",
			wantErr:  "Password and the confirmation do not match.",
		},
		{
			name:     "empty confirmation",
			password: "sust-250",
			confirm:  "",
			wantErr:  "Password and the confirmation do not match.",
		},
		{
			name:     "too long",
			password: strings.Repeat("x", MaxFieldLength+1),
			confirm:  strings.Repeat("x", MaxFieldLength+1),
			wantErr:  "Password must be at most 120 characters long.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Password("Password", tc.password, tc.confirm)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Password() returned an unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Password() returned no error, want %q", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("error = %q, want %q", err, tc.wantErr)
			}
		})
	}
}

func checkResult(t *testing.T, got string, err error, want, wantErr string) {
	t.Helper()
	if wantErr == "" {
		if err != nil {
			t.Fatalf("returned an unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("value = %q, want %q", got, want)
		}
		return
	}
	if err == nil {
		t.Fatalf("returned no error, want %q", wantErr)
	}
	if err.Error() != wantErr {
		t.Errorf("error = %q, want %q", err, wantErr)
	}
	if got != "" {
		t.Errorf("value = %q alongside the error, want an empty string", got)
	}
}
