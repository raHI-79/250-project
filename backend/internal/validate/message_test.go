package validate

import (
	"strings"
	"testing"
)

func TestMessageContent(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		hasFile bool
		want    string
		wantErr bool
	}{
		{name: "plain text", value: "  Bring the manual.  ", want: "Bring the manual."},
		{name: "empty with a file", value: "   ", hasFile: true},
		{name: "empty without a file", value: "   ", wantErr: true},
		{name: "at the limit", value: strings.Repeat("a", MaxMessageLength), want: strings.Repeat("a", MaxMessageLength)},
		{name: "over the limit", value: strings.Repeat("a", MaxMessageLength+1), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MessageContent("Content", tc.value, tc.hasFile)
			if (err != nil) != tc.wantErr {
				t.Fatalf("MessageContent() error = %v, want an error = %t", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("MessageContent() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMessageTargets(t *testing.T) {
	tests := []struct {
		name    string
		check   func(string, string) (string, error)
		value   string
		want    string
		wantErr bool
	}{
		{name: "own batch", check: CRMessageTarget, value: "OWN_BATCH", want: TargetOwnBatch},
		{name: "another batch", check: CRMessageTarget, value: "batch", want: TargetBatch},
		{name: "a teacher", check: CRMessageTarget, value: "teacher", want: TargetTeacher},
		{name: "a student is not a class representative target", check: CRMessageTarget, value: "student", wantErr: true},
		{name: "a class representative", check: TeacherMessageTarget, value: "cr", want: TargetCR},
		{name: "a batch", check: TeacherMessageTarget, value: "batch", want: TargetBatch},
		{name: "a student", check: TeacherMessageTarget, value: "student", want: TargetStudent},
		{name: "own batch is not a teacher target", check: TeacherMessageTarget, value: "own_batch", wantErr: true},
		{name: "a section poll", check: PollScope, value: "Section", want: PollScopeSection},
		{name: "a batch poll", check: PollScope, value: "batch", want: PollScopeBatch},
		{name: "an unknown scope", check: PollScope, value: "everyone", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.check("Target", tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("check(%q) error = %v, want an error = %t", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("check(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

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
