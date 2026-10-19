package validate

import (
	"strings"
	"testing"
)

func TestPollOptions(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    []string
		wantErr bool
	}{
		{name: "two options", values: []string{" Sunday ", "Monday"}, want: []string{"Sunday", "Monday"}},
		{name: "ten options", values: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			want: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}},
		{name: "one option", values: []string{"Sunday"}, wantErr: true},
		{name: "none", wantErr: true},
		{name: "eleven options", values: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}, wantErr: true},
		{name: "the same option twice", values: []string{"Sunday", "SUNDAY"}, wantErr: true},
		{name: "an empty option", values: []string{"Sunday", " "}, wantErr: true},
		{name: "a long option", values: []string{"Sunday", strings.Repeat("a", MaxFieldLength+1)}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PollOptions("Options", tc.values)
			if (err != nil) != tc.wantErr {
				t.Fatalf("PollOptions() error = %v, want an error = %t", err, tc.wantErr)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("PollOptions() = %v, want %v", got, tc.want)
			}
			for index := range got {
				if got[index] != tc.want[index] {
					t.Errorf("option %d = %q, want %q", index, got[index], tc.want[index])
				}
			}
		})
	}
}

func TestQuestion(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "a question", value: "  Which day? ", want: "Which day?"},
		{name: "empty", value: "   ", wantErr: true},
		{name: "too long", value: strings.Repeat("a", MaxQuestionLength+1), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Question("Question", tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Question() error = %v, want an error = %t", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("Question() = %q, want %q", got, tc.want)
			}
		})
	}
}
