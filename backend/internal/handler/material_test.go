package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const materialLink = "https://drive.google.com/file/d/abc123/view"

type fakeMaterialStore struct {
	profile      store.StudentProfile
	created      []store.NewMaterial
	materials    []store.MaterialRow
	askedBatch   string
	askedCreator int64
	deletedID    int64
	deletedBy    int64
	deleteErr    error
	err          error
}

func (f *fakeMaterialStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return f.profile, nil
}

func (f *fakeMaterialStore) CreateMaterial(ctx context.Context, material store.NewMaterial) (store.MaterialRow, error) {
	f.created = append(f.created, material)
	if f.err != nil {
		return store.MaterialRow{}, f.err
	}
	return store.MaterialRow{
		ID:            5,
		Batch:         material.Batch,
		Title:         material.Title,
		Link:          material.Link,
		CreatedByName: "Poster",
		CreatedAt:     time.Now(),
	}, nil
}

func (f *fakeMaterialStore) ListMaterials(ctx context.Context, batch string) ([]store.MaterialRow, error) {
	f.askedBatch = batch
	return f.materials, f.err
}

func (f *fakeMaterialStore) ListCreatorMaterials(ctx context.Context, createdBy int64) ([]store.MaterialRow, error) {
	f.askedCreator = createdBy
	return f.materials, f.err
}

func (f *fakeMaterialStore) ListStudentMaterials(ctx context.Context, batch string) ([]store.MaterialRow, error) {
	f.askedBatch = batch
	return f.materials, f.err
}

func (f *fakeMaterialStore) DeleteMaterial(ctx context.Context, materialID, createdBy int64) error {
	f.deletedID, f.deletedBy = materialID, createdBy
	return f.deleteErr
}

func jsonRequest(t *testing.T, method, path, body string, userID int64, role string) *http.Request {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: userID, Role: role}))
}

func TestMaterialCRCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBatch  string
		wantError  string
	}{
		{
			name:       "my batch only",
			body:       `{"title":"Slides","link":"` + materialLink + `","visibility":"mine"}`,
			wantStatus: http.StatusCreated,
			wantBatch:  noticeBatch,
		},
		{
			name:       "another batch",
			body:       `{"title":"Slides","link":"` + materialLink + `","visibility":"batch","batch":"cse-22"}`,
			wantStatus: http.StatusCreated,
			wantBatch:  otherBatch,
		},
		{
			name:       "all batches",
			body:       `{"title":"Slides","link":"` + materialLink + `","visibility":"all"}`,
			wantStatus: http.StatusCreated,
			wantBatch:  store.BatchAll,
		},
		{
			name:       "another batch without the batch code",
			body:       `{"title":"Slides","link":"` + materialLink + `","visibility":"batch"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageBatchForOneBatch,
		},
		{
			name:       "another batch that is the own batch",
			body:       `{"title":"Slides","link":"` + materialLink + `","visibility":"batch","batch":"CSE-23"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageOwnBatchMaterial,
		},
		{
			name:       "an unknown visibility",
			body:       `{"title":"Slides","link":"` + materialLink + `","visibility":"everyone"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Visibility must be one of mine, batch, all.",
		},
		{
			name:       "a link that is not a full url",
			body:       `{"title":"Slides","link":"drive.google.com/abc","visibility":"mine"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Link must be a full link that starts with http:// or https://.",
		},
		{
			name:       "a link with another scheme",
			body:       `{"title":"Slides","link":"ftp://files.sust.edu/slides.pdf","visibility":"mine"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Link must be a full link that starts with http:// or https://.",
		},
		{
			name:       "without a title",
			body:       `{"title":"  ","link":"` + materialLink + `","visibility":"mine"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Title is required.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			materials := &fakeMaterialStore{profile: store.StudentProfile{UserID: noticeCRID, Batch: noticeBatch, IsCR: true}}
			rec := httptest.NewRecorder()
			req := jsonRequest(t, http.MethodPost, "/api/cr/materials", tc.body, noticeCRID, auth.RoleCR)
			NewMaterial(materials).CRCreate()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if len(materials.created) != 0 {
					t.Errorf("the handler saved %d materials, want none", len(materials.created))
				}
				return
			}
			if len(materials.created) != 1 {
				t.Fatalf("the handler saved %d materials, want 1", len(materials.created))
			}
			saved := materials.created[0]
			if saved.Batch != tc.wantBatch || saved.CreatedBy != noticeCRID || saved.Link != materialLink {
				t.Errorf("saved material = %+v, want batch %q added by %d", saved, tc.wantBatch, noticeCRID)
			}
		})
	}
}

func TestMaterialAdminCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBatch  string
		wantError  string
	}{
		{
			name:       "one batch",
			body:       `{"batch":"cse-23","title":"Slides","link":"` + materialLink + `"}`,
			wantStatus: http.StatusCreated,
			wantBatch:  noticeBatch,
		},
		{
			name:       "all batches",
			body:       `{"batch":"all","title":"Slides","link":"` + materialLink + `"}`,
			wantStatus: http.StatusCreated,
			wantBatch:  store.BatchAll,
		},
		{
			name:       "a broken batch",
			body:       `{"batch":"23","title":"Slides","link":"` + materialLink + `"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "without a batch",
			body:       `{"title":"Slides","link":"` + materialLink + `"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch is required.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			materials := &fakeMaterialStore{}
			rec := httptest.NewRecorder()
			req := jsonRequest(t, http.MethodPost, "/api/admin/materials", tc.body, noticeAdminID, auth.RoleAdmin)
			NewMaterial(materials).AdminCreate()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if materials.created[0].Batch != tc.wantBatch {
				t.Errorf("batch = %q, want %q", materials.created[0].Batch, tc.wantBatch)
			}
		})
	}
}

func TestMaterialCRDelete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		deleteErr  error
		wantStatus int
	}{
		{name: "a material the class representative added", id: "5", wantStatus: http.StatusNoContent},
		{name: "a material of somebody else", id: "6", deleteErr: store.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "a broken id", id: "x", wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			materials := &fakeMaterialStore{
				profile:   store.StudentProfile{UserID: noticeCRID, Batch: noticeBatch, IsCR: true},
				deleteErr: tc.deleteErr,
			}
			req := jsonRequest(t, http.MethodDelete, "/api/cr/materials/"+tc.id, "", noticeCRID, auth.RoleCR)
			req.SetPathValue("id", tc.id)
			rec := httptest.NewRecorder()
			NewMaterial(materials).CRDelete()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus == http.StatusBadRequest {
				return
			}
			if materials.deletedBy != noticeCRID {
				t.Errorf("deleted on behalf of %d, want %d", materials.deletedBy, noticeCRID)
			}
		})
	}
}

func TestMaterialLists(t *testing.T) {
	rows := []store.MaterialRow{
		{ID: 2, Batch: store.BatchAll, Title: "Handbook", Link: materialLink, CreatedAt: time.Now()},
		{ID: 1, Batch: noticeBatch, Title: "Slides", Link: materialLink, CreatedAt: time.Now()},
	}
	tests := []struct {
		name         string
		path         string
		role         string
		pick         func(*Material) http.HandlerFunc
		wantBatch    string
		wantCreator  int64
		wantStatus   int
		wantFirstID  float64
		wantErrorMsg string
	}{
		{
			name:        "a student sees the own batch and ALL",
			path:        "/api/students/materials",
			role:        auth.RoleStudent,
			pick:        func(m *Material) http.HandlerFunc { return m.StudentList() },
			wantBatch:   noticeBatch,
			wantStatus:  http.StatusOK,
			wantFirstID: 2,
		},
		{
			name:        "a class representative sees what they added",
			path:        "/api/cr/materials",
			role:        auth.RoleCR,
			pick:        func(m *Material) http.HandlerFunc { return m.CRList() },
			wantCreator: noticeCRID,
			wantStatus:  http.StatusOK,
			wantFirstID: 2,
		},
		{
			name:        "the admin filters by batch",
			path:        "/api/admin/materials?batch=cse-22",
			role:        auth.RoleAdmin,
			pick:        func(m *Material) http.HandlerFunc { return m.AdminList() },
			wantBatch:   otherBatch,
			wantStatus:  http.StatusOK,
			wantFirstID: 2,
		},
		{
			name:         "the admin filter rejects a broken batch",
			path:         "/api/admin/materials?batch=22",
			role:         auth.RoleAdmin,
			pick:         func(m *Material) http.HandlerFunc { return m.AdminList() },
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "batch must look like CSE-23.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			materials := &fakeMaterialStore{
				profile:   store.StudentProfile{UserID: noticeCRID, Batch: noticeBatch, IsCR: true},
				materials: rows,
			}
			req := jsonRequest(t, http.MethodGet, tc.path, "", noticeCRID, tc.role)
			rec := httptest.NewRecorder()
			tc.pick(NewMaterial(materials))(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorOf(t, rec); got != tc.wantErrorMsg {
					t.Errorf("error = %q, want %q", got, tc.wantErrorMsg)
				}
				return
			}
			if materials.askedBatch != tc.wantBatch || materials.askedCreator != tc.wantCreator {
				t.Errorf("asked for batch %q and creator %d, want %q and %d",
					materials.askedBatch, materials.askedCreator, tc.wantBatch, tc.wantCreator)
			}
			items := noticeItems(t, rec)
			if len(items) != len(rows) || items[0]["id"] != tc.wantFirstID {
				t.Fatalf("items = %v, want %d entries starting with %v", items, len(rows), tc.wantFirstID)
			}
			if items[0]["link"] != materialLink || items[0]["batch"] != store.BatchAll {
				t.Errorf("first item = %v, want the ALL material with its link", items[0])
			}
		})
	}
}
