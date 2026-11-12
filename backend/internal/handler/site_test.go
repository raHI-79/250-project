package handler

import (
	"context"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/sitecontent"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeSiteStore struct {
	draft     *store.SiteVersion
	published []store.SiteVersion
	next      int64
}

func (f *fakeSiteStore) SaveSiteDraft(ctx context.Context, content []byte, userID int64) error {
	f.next++
	f.draft = &store.SiteVersion{ID: f.next, Kind: store.SiteDraft, Content: content, SavedBy: userID,
		SavedName: "chief", SavedAt: time.Now()}
	return nil
}

func (f *fakeSiteStore) SiteDraft(ctx context.Context) (store.SiteVersion, error) {
	if f.draft == nil {
		return store.SiteVersion{}, store.ErrNotFound
	}
	return *f.draft, nil
}

func (f *fakeSiteStore) SitePublished(ctx context.Context) (store.SiteVersion, error) {
	if len(f.published) == 0 {
		return store.SiteVersion{}, store.ErrNotFound
	}
	return f.published[len(f.published)-1], nil
}

func (f *fakeSiteStore) SiteVersion(ctx context.Context, id int64) (store.SiteVersion, error) {
	for _, version := range f.published {
		if version.ID == id {
			return version, nil
		}
	}
	return store.SiteVersion{}, store.ErrNotFound
}

func (f *fakeSiteStore) PublishSite(ctx context.Context, content []byte, userID int64, note string) (int64, error) {
	f.next++
	f.published = append(f.published, store.SiteVersion{ID: f.next, Kind: store.SitePublished,
		Content: content, SavedBy: userID, SavedName: "chief", SavedAt: time.Now(), Note: note})
	return f.next, nil
}

func (f *fakeSiteStore) ListSiteVersions(ctx context.Context, limit int) ([]store.SiteVersion, error) {
	listed := make([]store.SiteVersion, 0, len(f.published))
	for index := len(f.published) - 1; index >= 0; index-- {
		listed = append(listed, f.published[index])
	}
	return listed, nil
}

var siteChief = &middleware.User{ID: 1, Identifier: "chief", Role: "super_admin"}

func siteCall(t *testing.T, handlerFunc http.HandlerFunc, method, target, body string,
	user *middleware.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req = req.WithContext(middleware.WithUser(req.Context(), *user))
	}
	if id := strings.TrimPrefix(target, "/api/super-admin/site/versions/"); id != target {
		req.SetPathValue("id", strings.TrimSuffix(id, "/restore"))
	}
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func siteAnswer(t *testing.T, rec *httptest.ResponseRecorder) siteBody {
	t.Helper()
	var body siteBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("the answer is not website content: %v (%s)", err, rec.Body.String())
	}
	return body
}

func named(t *testing.T, name string) string {
	t.Helper()
	content := sitecontent.Default()
	content.Header.SiteName = name
	raw, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("could not encode the content: %v", err)
	}
	return string(raw)
}

func TestThePublicWebsiteFallsBackToTheBuiltInContent(t *testing.T) {
	site := NewSite(nil, &fakeSiteStore{}, nil, nil)
	rec := siteCall(t, site.Public(), http.MethodGet, "/api/site/content", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := siteAnswer(t, rec)
	if !body.Fallback || body.Version != 0 || body.Content.Header.SiteName != "SUST CSE" {
		t.Fatalf("nothing is saved but the answer is %+v", body)
	}
	if body.Developer.Label != "Developed by CoordiNet" || body.Developer.Path != "/about-coordinet" {
		t.Fatalf("the fixed credit is %+v", body.Developer)
	}
	broken := &fakeSiteStore{published: []store.SiteVersion{{ID: 4, Kind: store.SitePublished,
		Content: []byte(`{"header":{"site_name":""}}`)}}}
	rec = siteCall(t, NewSite(nil, broken, nil, nil).Public(), http.MethodGet, "/api/site/content", "", nil)
	if body := siteAnswer(t, rec); !body.Fallback {
		t.Fatalf("broken saved content was served: %+v", body.Content.Header)
	}
}

func TestADraftStaysPrivateUntilItIsPublished(t *testing.T) {
	saved := &fakeSiteStore{}
	site := NewSite(nil, saved, nil, nil)
	rec := siteCall(t, site.SaveDraft(), http.MethodPut, "/api/super-admin/site/draft",
		named(t, "CSE at SUST"), siteChief)
	if rec.Code != http.StatusOK {
		t.Fatalf("saving a draft answered %d: %s", rec.Code, rec.Body.String())
	}
	public := siteAnswer(t, siteCall(t, site.Public(), http.MethodGet, "/api/site/content", "", nil))
	if public.Content.Header.SiteName != "SUST CSE" || !public.Fallback {
		t.Fatalf("an unpublished draft is public: %+v", public.Content.Header)
	}
	preview := siteAnswer(t, siteCall(t, site.Draft(), http.MethodGet, "/api/super-admin/site/draft", "", siteChief))
	if preview.Content.Header.SiteName != "CSE at SUST" || !preview.Draft {
		t.Fatalf("the preview does not show the draft: %+v", preview)
	}
	rec = siteCall(t, site.Publish(), http.MethodPost, "/api/super-admin/site/publish", "", siteChief)
	if rec.Code != http.StatusOK {
		t.Fatalf("publishing answered %d: %s", rec.Code, rec.Body.String())
	}
	public = siteAnswer(t, siteCall(t, site.Public(), http.MethodGet, "/api/site/content", "", nil))
	if public.Content.Header.SiteName != "CSE at SUST" || public.Fallback || public.Version == 0 {
		t.Fatalf("the published content is not public: %+v", public)
	}
	if public.SavedBy != "chief" || public.SavedAt == "" {
		t.Fatalf("the published version does not say who and when: %+v", public)
	}
}

func TestSavingRefusesBadContentAndTheCredit(t *testing.T) {
	site := NewSite(nil, &fakeSiteStore{}, nil, nil)
	credit := strings.Replace(named(t, "X"), `"footer":{`, `"footer":{"hide_credit":true,`, 1)
	rec := siteCall(t, site.SaveDraft(), http.MethodPut, "/api/super-admin/site/draft", credit, siteChief)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "developer credit") {
		t.Fatalf("hiding the credit answered %d: %s", rec.Code, rec.Body.String())
	}
	rec = siteCall(t, site.SaveDraft(), http.MethodPut, "/api/super-admin/site/draft", `{"header":{}}`, siteChief)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("content without a site name answered %d", rec.Code)
	}
	rec = siteCall(t, site.SaveDraft(), http.MethodPut, "/api/super-admin/site/draft", named(t, "X"), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("saving without signing in answered %d", rec.Code)
	}
	rec = siteCall(t, site.Publish(), http.MethodPost, "/api/super-admin/site/publish", "", siteChief)
	if rec.Code != http.StatusConflict {
		t.Fatalf("publishing without a draft answered %d", rec.Code)
	}
}

func TestEveryPublishedVersionIsKeptAndCanBeRestored(t *testing.T) {
	saved := &fakeSiteStore{}
	site := NewSite(nil, saved, nil, nil)
	for _, name := range []string{"First name", "Second name"} {
		siteCall(t, site.SaveDraft(), http.MethodPut, "/api/super-admin/site/draft", named(t, name), siteChief)
		rec := siteCall(t, site.Publish(), http.MethodPost, "/api/super-admin/site/publish", "", siteChief)
		if rec.Code != http.StatusOK {
			t.Fatalf("publishing %q answered %d", name, rec.Code)
		}
	}
	var listed siteVersionsBody
	rec := siteCall(t, site.Versions(), http.MethodGet, "/api/super-admin/site/versions", "", siteChief)
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || len(listed.Items) != 2 {
		t.Fatalf("the versions are %s, %v", rec.Body.String(), err)
	}
	if !listed.Items[0].Live || listed.Items[1].Live || listed.Items[0].SavedBy != "chief" {
		t.Fatalf("the versions do not say which is live and who saved them: %+v", listed.Items)
	}
	first := listed.Items[1].ID
	rec = siteCall(t, site.Restore(), http.MethodPost,
		"/api/super-admin/site/versions/"+strings.TrimSpace(jsonNumber(first))+"/restore", "", siteChief)
	if rec.Code != http.StatusOK {
		t.Fatalf("restoring answered %d: %s", rec.Code, rec.Body.String())
	}
	public := siteAnswer(t, siteCall(t, site.Public(), http.MethodGet, "/api/site/content", "", nil))
	if public.Content.Header.SiteName != "First name" {
		t.Fatalf("after the restore the site is called %q", public.Content.Header.SiteName)
	}
	if len(saved.published) != 3 || !strings.Contains(saved.published[2].Note, "Restored version") {
		t.Fatalf("the restore did not keep its own version: %+v", saved.published)
	}
	draft := siteAnswer(t, siteCall(t, site.Draft(), http.MethodGet, "/api/super-admin/site/draft", "", siteChief))
	if draft.Content.Header.SiteName != "First name" {
		t.Fatalf("the draft still says %q after the restore", draft.Content.Header.SiteName)
	}
	rec = siteCall(t, site.Restore(), http.MethodPost, "/api/super-admin/site/versions/999/restore", "", siteChief)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("restoring a missing version answered %d", rec.Code)
	}
}

func jsonNumber(value int64) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
