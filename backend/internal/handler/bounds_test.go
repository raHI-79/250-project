package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
)

func TestAListRefusesAPageSizeOutsideItsRange(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name  string
		limit string
		want  int
	}{
		{name: "no limit at all", limit: "", want: http.StatusOK},
		{name: "the smallest page", limit: "1", want: http.StatusOK},
		{name: "the largest page", limit: "100", want: http.StatusOK},
		{name: "one past the largest page", limit: "101", want: http.StatusBadRequest},
		{name: "a thousand rows", limit: "1000", want: http.StatusBadRequest},
		{name: "no rows", limit: "0", want: http.StatusBadRequest},
		{name: "a negative page", limit: "-5", want: http.StatusBadRequest},
		{name: "not a number", limit: "all", want: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeNotificationService{}
			handler := NewNotifications(service)
			path := "/api/notifications"
			if test.limit != "" {
				path += "?limit=" + test.limit
			}
			request := httptest.NewRequest(http.MethodGet, path, nil)
			rec := serve(handler.List(), withCurrentUser(request, 7, auth.RoleStudent))
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.want, rec.Body.String())
			}
			if test.want != http.StatusOK {
				if got := decodeBody(t, rec)["error"]; got != MessageInvalidNotificationLimit {
					t.Errorf("error = %v, want %q", got, MessageInvalidNotificationLimit)
				}
				return
			}
			if service.askedLimit > notify.MaxLimit {
				t.Errorf("the store was asked for %d rows, want at most %d",
					service.askedLimit, notify.MaxLimit)
			}
			if service.askedLimit <= 0 {
				t.Errorf("the store was asked for %d rows, want a page", service.askedLimit)
			}
		})
	}
}

func TestTheDefaultPageIsAlwaysBounded(t *testing.T) {
	silenceLog(t)
	service := &fakeNotificationService{}
	request := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	rec := serve(NewNotifications(service).List(), withCurrentUser(request, 7, auth.RoleStudent))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if service.askedLimit != notify.DefaultLimit {
		t.Errorf("the store was asked for %d rows, want the standing page of %d",
			service.askedLimit, notify.DefaultLimit)
	}
	if notify.DefaultLimit <= 0 || notify.DefaultLimit > notify.MaxLimit {
		t.Errorf("the standing page is %d, which is not a page at all", notify.DefaultLimit)
	}
}

func TestTheStoreBoundsTheListsACallerCannotNarrow(t *testing.T) {
	bounds := map[string]int{
		"the student directory":  store.DirectoryLimit,
		"the message contacts":   store.ContactsLimit,
		"the drop course list":   store.DropCourseOptionLimit,
		"the activity log page":  store.DefaultAuditLimit,
		"a page of chat history": store.MessagePageSize,
	}
	for name, bound := range bounds {
		if bound <= 0 {
			t.Errorf("%s has no bound", name)
		}
		if bound > 1000 {
			t.Errorf("%s stops at %d rows, which is not a bound worth having", name, bound)
		}
	}
}
