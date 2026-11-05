package validate

import "testing"

func TestOfficePasswordNeedsLengthLetterAndDigit(t *testing.T) {
	tests := []struct {
		name     string
		password string
		confirm  string
		wants    string
	}{
		{name: "empty", password: "", confirm: "", wants: "New password is required."},
		{
			name:     "too short",
			password: "short1",
			confirm:  "short1",
			wants:    "New password must be at least 10 characters long.",
		},
		{
			name:     "letters only",
			password: "onlyletters",
			confirm:  "onlyletters",
			wants:    "New password must contain a letter and a digit.",
		},
		{
			name:     "digits only",
			password: "1234567890",
			confirm:  "1234567890",
			wants:    "New password must contain a letter and a digit.",
		},
		{
			name:     "mismatch",
			password: "office2026x",
			confirm:  "office2026y",
			wants:    "New password and the confirmation do not match.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := OfficePassword("New password", tc.password, tc.confirm)
			if err == nil {
				t.Fatalf("OfficePassword(%q) accepted it, want %q", tc.password, tc.wants)
			}
			if err.Error() != tc.wants {
				t.Errorf("error = %q, want %q", err.Error(), tc.wants)
			}
		})
	}
	if err := OfficePassword("New password", "office2026x", "office2026x"); err != nil {
		t.Errorf("a good password was refused: %v", err)
	}
	if MinOfficePasswordLength < 10 {
		t.Errorf("MinOfficePasswordLength = %d, want at least 10", MinOfficePasswordLength)
	}
}
