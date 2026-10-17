package validate

import (
	"strings"
	"testing"
)

func TestLink(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "https", value: " https://drive.google.com/d/abc ", want: "https://drive.google.com/d/abc"},
		{name: "http", value: "http://sust.edu/notice.pdf", want: "http://sust.edu/notice.pdf"},
		{name: "without a scheme", value: "drive.google.com/d/abc", wantErr: true},
		{name: "another scheme", value: "ftp://sust.edu/notice.pdf", wantErr: true},
		{name: "without a host", value: "https://", wantErr: true},
		{name: "empty", value: "   ", wantErr: true},
		{name: "too long", value: "https://sust.edu/" + strings.Repeat("a", MaxLinkLength), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Link("Link", tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Link(%q) error = %v, want an error = %t", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("Link(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestBatchOrAll(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "a batch code", value: "cse-23", want: "CSE-23"},
		{name: "all batches", value: "all", want: BatchAll},
		{name: "already upper case", value: "ALL", want: BatchAll},
		{name: "a broken code", value: "23", wantErr: true},
		{name: "empty", value: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BatchOrAll("Batch", tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("BatchOrAll(%q) error = %v, want an error = %t", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("BatchOrAll(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestVisibilityAndScope(t *testing.T) {
	tests := []struct {
		name    string
		check   func(string, string) (string, error)
		value   string
		want    string
		wantErr bool
	}{
		{name: "mine", check: Visibility, value: "Mine", want: VisibilityMine},
		{name: "batch", check: Visibility, value: "batch", want: VisibilityBatch},
		{name: "all", check: Visibility, value: "all", want: VisibilityAll},
		{name: "an unknown visibility", check: Visibility, value: "everyone", wantErr: true},
		{name: "my notices", check: NoticeScope, value: "MY", want: NoticeScopeMine},
		{name: "other notices", check: NoticeScope, value: "other", want: NoticeScopeOthers},
		{name: "an unknown scope", check: NoticeScope, value: "others", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.check("Value", tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("check(%q) error = %v, want an error = %t", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("check(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestDescriptionAndSearch(t *testing.T) {
	tests := []struct {
		name    string
		check   func(string, string) (string, error)
		value   string
		want    string
		wantErr bool
	}{
		{name: "a description", check: Description, value: "  Pay the fees.  ", want: "Pay the fees."},
		{name: "an empty description", check: Description, value: "   "},
		{name: "a long description", check: Description, value: strings.Repeat("a", MaxDescription+1), wantErr: true},
		{name: "a search", check: Search, value: " Ratul ", want: "Ratul"},
		{name: "an empty search", check: Search, value: ""},
		{name: "a long search", check: Search, value: strings.Repeat("a", MaxSearchLength+1), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.check("Value", tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("check(%q) error = %v, want an error = %t", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("check(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
