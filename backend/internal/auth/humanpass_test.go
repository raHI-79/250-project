package auth

import (
	"testing"
	"time"
)

const humanSecret = "a-long-enough-secret-for-the-human-pass"

func TestHumanPassRoundTrip(t *testing.T) {
	passes, err := NewHumanPasses(humanSecret)
	if err != nil {
		t.Fatalf("NewHumanPasses() returned %v", err)
	}
	token, expiresAt, err := passes.Issue(HumanPassBrowser)
	if err != nil {
		t.Fatalf("Issue() returned %v", err)
	}
	if time.Until(expiresAt) <= 0 {
		t.Errorf("expiresAt = %v, want it in the future", expiresAt)
	}
	if err := passes.Check(token); err != nil {
		t.Errorf("Check() returned %v, want no error", err)
	}
}

func TestHumanPassRefusesTheWrongToken(t *testing.T) {
	passes, err := NewHumanPasses(humanSecret)
	if err != nil {
		t.Fatalf("NewHumanPasses() returned %v", err)
	}
	other, err := NewHumanPasses(humanSecret + "-other")
	if err != nil {
		t.Fatalf("NewHumanPasses() returned %v", err)
	}
	foreign, _, err := other.Issue(HumanPassBrowser)
	if err != nil {
		t.Fatalf("Issue() returned %v", err)
	}
	for _, token := range []string{"", "not-a-token", foreign} {
		if err := passes.Check(token); err == nil {
			t.Errorf("Check(%q) accepted a token it should refuse", token)
		}
	}
}

func TestHumanPassExpires(t *testing.T) {
	passes, err := NewHumanPasses(humanSecret)
	if err != nil {
		t.Fatalf("NewHumanPasses() returned %v", err)
	}
	token, _, err := passes.Issue(time.Minute)
	if err != nil {
		t.Fatalf("Issue() returned %v", err)
	}
	passes.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if err := passes.Check(token); err == nil {
		t.Error("Check() accepted an expired pass")
	}
}

func TestHumanPassNeedsAStrongSecret(t *testing.T) {
	if _, err := NewHumanPasses("short"); err == nil {
		t.Error("NewHumanPasses() accepted a short secret")
	}
}

func TestHumanPassRefusesABadTTL(t *testing.T) {
	passes, err := NewHumanPasses(humanSecret)
	if err != nil {
		t.Fatalf("NewHumanPasses() returned %v", err)
	}
	if _, _, err := passes.Issue(0); err == nil {
		t.Error("Issue() accepted a zero lifetime")
	}
}
