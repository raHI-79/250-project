package server

import (
	"net/http"
	"testing"
)

func TestPushSubscriptionFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	onboardStudent(t, router, adminToken, chatFlowStudent)
	studentToken := loginToken(t, router, "2023331961")
	key := call(t, router, http.MethodGet, "/api/push/public-key", "", "")
	if key.Code != http.StatusOK {
		t.Fatalf("public key status = %d, want 200 (body = %q)", key.Code, key.Body.String())
	}
	if got := field(t, key, "public_key"); got != "test-public-key" {
		t.Errorf("public_key = %q, want the configured key", got)
	}
	body := `{"endpoint":"https://push.example/flow","keys":{"p256dh":"flow-key","auth":"flow-secret"},
"user_agent":"Chrome on Android"}`
	saved := call(t, router, http.MethodPost, "/api/push/subscribe", body, studentToken)
	if saved.Code != http.StatusCreated {
		t.Fatalf("subscribe status = %d, want 201 (body = %q)", saved.Code, saved.Body.String())
	}
	var stored int
	query := "SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = $1"
	if err := pool.QueryRow(ctx, query, "https://push.example/flow").Scan(&stored); err != nil {
		t.Fatalf("could not read back the subscription: %v", err)
	}
	if stored != 1 {
		t.Fatalf("%d subscriptions stored, want 1", stored)
	}
	anonymous := call(t, router, http.MethodPost, "/api/push/subscribe", body, "")
	if anonymous.Code != http.StatusUnauthorized {
		t.Errorf("subscribing without a token status = %d, want 401", anonymous.Code)
	}
	office := call(t, router, http.MethodPost, "/api/push/subscribe", body, adminToken)
	if office.Code != http.StatusForbidden {
		t.Errorf("the office subscribing status = %d, want 403", office.Code)
	}
	removed := call(t, router, http.MethodDelete, "/api/push/subscribe",
		`{"endpoint":"https://push.example/flow"}`, studentToken)
	if removed.Code != http.StatusOK {
		t.Fatalf("unsubscribe status = %d, want 200 (body = %q)", removed.Code, removed.Body.String())
	}
	if err := pool.QueryRow(ctx, query, "https://push.example/flow").Scan(&stored); err != nil {
		t.Fatalf("could not read back the subscription: %v", err)
	}
	if stored != 0 {
		t.Errorf("%d subscriptions stayed, want none", stored)
	}
}
