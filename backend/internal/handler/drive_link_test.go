package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	driveBatch    = "CSE-23"
	driveOther    = "CSE-22"
	driveCRID     = int64(71)
	driveLinkHTTP = "https://drive.google.com/drive/folders/cse23"
)

type fakeDriveLinkStore struct {
	profile store.StudentProfile
	links   map[string]store.DriveLinkRow
	saved   []store.NewDriveLink
	removed []string
	err     error
}

func newDriveLinkStore(isCR bool) *fakeDriveLinkStore {
	return &fakeDriveLinkStore{
		profile: store.StudentProfile{UserID: driveCRID, StudentID: 4, Batch: driveBatch, Section: "A", IsCR: isCR},
		links:   make(map[string]store.DriveLinkRow),
	}
}

func (f *fakeDriveLinkStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return f.profile, nil
}

func (f *fakeDriveLinkStore) GetBatchDriveLink(ctx context.Context, batch string) (store.DriveLinkRow, error) {
	if f.err != nil {
		return store.DriveLinkRow{}, f.err
	}
	row, ok := f.links[batch]
	if !ok {
		return store.DriveLinkRow{}, store.ErrNotFound
	}
	return row, nil
}

func (f *fakeDriveLinkStore) SaveBatchDriveLink(ctx context.Context, link store.NewDriveLink) (store.DriveLinkRow, error) {
	f.saved = append(f.saved, link)
	if f.err != nil {
		return store.DriveLinkRow{}, f.err
	}
	row := store.DriveLinkRow{Batch: link.Batch, Title: link.Title, Link: link.Link, UpdatedAt: time.Now()}
	f.links[link.Batch] = row
	return row, nil
}

func (f *fakeDriveLinkStore) DeleteBatchDriveLink(ctx context.Context, batch string) error {
	f.removed = append(f.removed, batch)
	if f.err != nil {
		return f.err
	}
	if _, ok := f.links[batch]; !ok {
		return store.ErrNotFound
	}
	delete(f.links, batch)
	return nil
}

func (f *fakeDriveLinkStore) ListBatchDriveLinks(ctx context.Context) ([]store.DriveLinkRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []store.DriveLinkRow{
		{Batch: driveOther, Title: driveOther, Link: driveLinkHTTP, UpdatedAt: time.Now()},
		{Batch: driveBatch, Title: driveBatch, Link: driveLinkHTTP, UpdatedAt: time.Now()},
	}, nil
}

func driveCall(t *testing.T, handlerFunc http.HandlerFunc, method, body string, role string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := jsonRequest(t, method, "/api/cr/drive-link", body, driveCRID, role)
	handlerFunc(rec, req)
	return rec
}

func TestDriveLinkCRSave(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		isCR       bool
		wantStatus int
		wantError  string
	}{
		{name: "a full link", body: `{"link":"` + driveLinkHTTP + `"}`, isCR: true, wantStatus: http.StatusOK},
		{
			name:       "a link without a scheme",
			body:       `{"link":"drive.google.com/cse23"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Link must be a full link that starts with http:// or https://.",
		},
		{
			name:       "a link with another scheme",
			body:       `{"link":"ftp://files.sust.edu/cse23"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Link must be a full link that starts with http:// or https://.",
		},
		{
			name:       "an empty link",
			body:       `{"link":"   "}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Link is required.",
		},
		{
			name:       "a student who is not a class representative",
			body:       `{"link":"` + driveLinkHTTP + `"}`,
			isCR:       false,
			wantStatus: http.StatusForbidden,
			wantError:  MessageNotACR,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			links := newDriveLinkStore(test.isCR)
			rec := driveCall(t, NewDriveLink(links).CRSave(), http.MethodPut, test.body, auth.RoleCR)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				if got := errorOf(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(links.saved) != 0 {
					t.Errorf("the handler saved %d links, want none", len(links.saved))
				}
				return
			}
			if len(links.saved) != 1 {
				t.Fatalf("the handler saved %d links, want 1", len(links.saved))
			}
			saved := links.saved[0]
			if saved.Batch != driveBatch || saved.Title != driveBatch || saved.UpdatedBy != driveCRID {
				t.Errorf("saved link = %+v, want the own batch as the title", saved)
			}
			body := decodeBody(t, rec)
			if body["batch"] != driveBatch || body["title"] != driveBatch || body["is_mine"] != true {
				t.Errorf("payload = %v, want the own batch marked as mine", body)
			}
		})
	}
}

func TestDriveLinkCRGetAndDelete(t *testing.T) {
	silenceLog(t)
	links := newDriveLinkStore(true)
	empty := driveCall(t, NewDriveLink(links).CRGet(), http.MethodGet, "", auth.RoleCR)
	if empty.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", empty.Code, empty.Body.String())
	}
	if body := decodeBody(t, empty); body["link"] != "" || body["title"] != driveBatch {
		t.Errorf("payload = %v, want an empty link titled with the batch code", body)
	}
	missing := driveCall(t, NewDriveLink(links).CRDelete(), http.MethodDelete, "", auth.RoleCR)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("delete without a link = %d, want 404 (body = %q)", missing.Code, missing.Body.String())
	}
	if got := errorOf(t, missing); got != MessageDriveLinkMissing {
		t.Errorf("error = %q, want %q", got, MessageDriveLinkMissing)
	}
	saved := driveCall(t, NewDriveLink(links).CRSave(), http.MethodPut, `{"link":"`+driveLinkHTTP+`"}`, auth.RoleCR)
	if saved.Code != http.StatusOK {
		t.Fatalf("save status = %d, want 200 (body = %q)", saved.Code, saved.Body.String())
	}
	current := driveCall(t, NewDriveLink(links).CRGet(), http.MethodGet, "", auth.RoleCR)
	if body := decodeBody(t, current); body["link"] != driveLinkHTTP {
		t.Errorf("payload = %v, want the saved link", body)
	}
	removed := driveCall(t, NewDriveLink(links).CRDelete(), http.MethodDelete, "", auth.RoleCR)
	if removed.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204 (body = %q)", removed.Code, removed.Body.String())
	}
	if len(links.removed) != 2 || links.removed[1] != driveBatch {
		t.Errorf("the store was asked to remove %v, want the own batch", links.removed)
	}
}

func TestDriveLinkStudentList(t *testing.T) {
	silenceLog(t)
	links := newDriveLinkStore(false)
	rec := driveCall(t, NewDriveLink(links).StudentList(), http.MethodGet, "", auth.RoleStudent)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	list := items(t, rec)
	if len(list) != 2 {
		t.Fatalf("the list has %d cards, want 2", len(list))
	}
	if list[0]["batch"] != driveOther || list[0]["is_mine"] != false {
		t.Errorf("first card = %v, want the other batch", list[0])
	}
	if list[1]["batch"] != driveBatch || list[1]["is_mine"] != true {
		t.Errorf("second card = %v, want the own batch marked as mine", list[1])
	}
}
