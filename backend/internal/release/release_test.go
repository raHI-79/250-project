package release

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeAPI struct {
	server  *httptest.Server
	calls   int
	status  int
	body    any
	raw     string
	headers map[string]string
	seen    http.Header
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	fake := &fakeAPI{status: http.StatusOK, headers: map[string]string{}}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.calls++
		fake.seen = r.Header.Clone()
		if r.URL.Path != "/repos/raHI-79/250-project/releases" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		for key, value := range fake.headers {
			w.Header().Set(key, value)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(fake.status)
		if fake.raw != "" {
			_, _ = w.Write([]byte(fake.raw))
			return
		}
		_ = json.NewEncoder(w).Encode(fake.body)
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeAPI) lookup(clock *time.Time) *Lookup {
	return NewWith("", f.server.URL, f.server.Client(), func() time.Time { return *clock })
}

func releaseEntry(tag, notes string, assets ...string) map[string]any {
	listed := make([]map[string]any, 0, len(assets))
	for _, name := range assets {
		listed = append(listed, map[string]any{
			"name":                 name,
			"browser_download_url": "https://example.test/" + name,
		})
	}
	return map[string]any{
		"tag_name": tag,
		"body":     notes,
		"draft":    false,
		"assets":   listed,
	}
}

func releaseBody(tag, notes, assetName string) []map[string]any {
	return []map[string]any{releaseEntry(tag, notes, "source.zip", assetName)}
}

func TestLatestReadsTheNewestAPKAsset(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v2.1.0", "Fixes the routine", "sust-cse-portal-2.1.0.apk")
	clock := time.Now()
	details, err := fake.lookup(&clock).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if details.Version != "2.1.0" {
		t.Errorf("Version = %q, want %q", details.Version, "2.1.0")
	}
	if details.APKURL != "https://example.test/sust-cse-portal-2.1.0.apk" {
		t.Errorf("APKURL = %q", details.APKURL)
	}
	if details.Notes != "Fixes the routine" {
		t.Errorf("Notes = %q", details.Notes)
	}
	if details.Required {
		t.Error("Required = true, want false")
	}
	if details.VersionCode != VersionCode("2.1.0") {
		t.Errorf("VersionCode = %d, want %d", details.VersionCode, VersionCode("2.1.0"))
	}
}

func TestLatestSendsTheHeadersGitHubWants(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v2.1.0", "", "sust-cse-portal-2.1.0.apk")
	clock := time.Now()
	lookup := fake.lookup(&clock).WithToken("super-secret-token")
	if _, err := lookup.Latest(context.Background()); err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if got := fake.seen.Get("User-Agent"); got != userAgent {
		t.Errorf("User-Agent = %q, want %q", got, userAgent)
	}
	if got := fake.seen.Get("Authorization"); got != "Bearer super-secret-token" {
		t.Errorf("Authorization = %q, want the bearer token", got)
	}
	if got := fake.seen.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("Accept = %q", got)
	}
}

func TestLatestStaysAnonymousWithoutAToken(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v2.1.0", "", "sust-cse-portal-2.1.0.apk")
	clock := time.Now()
	if _, err := fake.lookup(&clock).Latest(context.Background()); err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if got := fake.seen.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want no header at all", got)
	}
	if got := fake.seen.Get("User-Agent"); got != userAgent {
		t.Errorf("User-Agent = %q, want %q", got, userAgent)
	}
}

func TestLatestSkipsTheNewestReleaseWithoutTheAPK(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = []map[string]any{
		releaseEntry("v2.2.0", "Attendance by Bluetooth"),
		releaseEntry("v2.1.0", "Adds in-app updates", "sust-cse-portal-2.1.0.apk"),
	}
	clock := time.Now()
	details, err := fake.lookup(&clock).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if details.Version != "2.1.0" {
		t.Errorf("Version = %q, want 2.1.0 because v2.2.0 has no APK", details.Version)
	}
	if details.APKURL != "https://example.test/sust-cse-portal-2.1.0.apk" {
		t.Errorf("APKURL = %q", details.APKURL)
	}
}

func TestLatestSkipsDraftsPrereleasesAndMismatchedAssets(t *testing.T) {
	draft := releaseEntry("v3.0.0", "", "sust-cse-portal-3.0.0.apk")
	draft["draft"] = true
	early := releaseEntry("v2.9.0", "", "sust-cse-portal-2.9.0.apk")
	early["prerelease"] = true
	wrong := releaseEntry("v2.5.0", "", "sust-cse-portal-2.4.0.apk")
	fake := newFakeAPI(t)
	fake.body = []map[string]any{
		draft,
		early,
		wrong,
		releaseEntry("v2.1.0", "", "sust-cse-portal-2.1.0.apk"),
	}
	clock := time.Now()
	details, err := fake.lookup(&clock).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if details.Version != "2.1.0" {
		t.Errorf("Version = %q, want 2.1.0", details.Version)
	}
}

func TestLatestMarksAReleaseRequired(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v3.0.0", "[REQUIRED] sign in again", "sust-cse-portal-3.0.0.apk")
	clock := time.Now()
	details, err := fake.lookup(&clock).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if !details.Required {
		t.Error("Required = false, want true")
	}
}

func TestLatestReportsNoReleaseWithAnAPK(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = []map[string]any{releaseEntry("v2.1.0", "")}
	clock := time.Now()
	details, err := fake.lookup(&clock).Latest(context.Background())
	if !errors.Is(err, ErrNoAsset) {
		t.Fatalf("Latest() error = %v, want ErrNoAsset", err)
	}
	if details.APKURL != "" || details.Version != "" {
		t.Errorf("details = %+v, want an empty answer", details)
	}
}

func TestLatestAnswersFromTheCacheWithinTenMinutes(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v2.1.0", "", "sust-cse-portal-2.1.0.apk")
	clock := time.Now()
	lookup := fake.lookup(&clock)
	for i := 0; i < 3; i++ {
		if _, err := lookup.Latest(context.Background()); err != nil {
			t.Fatalf("Latest() error = %v", err)
		}
	}
	if fake.calls != 1 {
		t.Fatalf("GitHub was asked %d times, want 1", fake.calls)
	}
	clock = clock.Add(CacheTTL + time.Minute)
	if _, err := lookup.Latest(context.Background()); err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if fake.calls != 2 {
		t.Errorf("GitHub was asked %d times after the cache expired, want 2", fake.calls)
	}
}

func TestLatestKeepsTheLastGoodAnswerThroughARateLimit(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v2.1.0", "Fixes", "sust-cse-portal-2.1.0.apk")
	clock := time.Now()
	lookup := fake.lookup(&clock)
	first, err := lookup.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	fake.status = http.StatusForbidden
	fake.raw = `{"message":"API rate limit exceeded for 10.0.0.1"}`
	fake.headers["X-RateLimit-Remaining"] = "0"
	start := clock
	tests := []struct {
		name  string
		after time.Duration
		want  Details
		err   error
	}{
		{name: "an hour later", after: time.Hour, want: first},
		{name: "five hours later", after: 5 * time.Hour, want: first},
		{name: "six hours later", after: StaleTTL + time.Minute, err: ErrUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clock = start.Add(tc.after)
			details, err := lookup.Latest(context.Background())
			if !errors.Is(err, tc.err) {
				t.Fatalf("Latest() error = %v, want %v", err, tc.err)
			}
			if details != tc.want {
				t.Errorf("details = %+v, want %+v", details, tc.want)
			}
		})
	}
}

func TestLatestGivesUpWhenARateLimitHitsAnEmptyCache(t *testing.T) {
	fake := newFakeAPI(t)
	fake.status = http.StatusTooManyRequests
	fake.raw = `{"message":"API rate limit exceeded"}`
	fake.headers["Retry-After"] = "60"
	clock := time.Now()
	lookup := fake.lookup(&clock)
	details, err := lookup.Latest(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Latest() error = %v, want ErrUnavailable", err)
	}
	if details != (Details{}) {
		t.Errorf("details = %+v, want an empty answer", details)
	}
	if _, err := lookup.Latest(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Latest() error = %v, want ErrUnavailable", err)
	}
	if fake.calls != 1 {
		t.Errorf("GitHub was asked %d times inside the retry window, want 1", fake.calls)
	}
	clock = clock.Add(RetryAfter + time.Second)
	fake.status = http.StatusOK
	fake.raw = ""
	fake.headers = map[string]string{}
	fake.body = releaseBody("v2.1.0", "", "sust-cse-portal-2.1.0.apk")
	details, err = lookup.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v once GitHub recovered", err)
	}
	if details.Version != "2.1.0" {
		t.Errorf("Version = %q, want 2.1.0", details.Version)
	}
}

func TestLatestSurvivesEveryGitHubFailure(t *testing.T) {
	tests := []struct {
		name   string
		status int
		raw    string
	}{
		{name: "rate limited", status: http.StatusForbidden, raw: `{"message":"rate limit"}`},
		{name: "too many requests", status: http.StatusTooManyRequests, raw: `{"message":"slow down"}`},
		{name: "server error", status: http.StatusInternalServerError, raw: `{"message":"boom"}`},
		{name: "bad gateway", status: http.StatusBadGateway, raw: ``},
		{name: "bad json", status: http.StatusOK, raw: `{"this is not": a list`},
		{name: "html instead of json", status: http.StatusOK, raw: `<html>maintenance</html>`},
		{name: "an object instead of a list", status: http.StatusOK, raw: `{"tag_name":"v2.1.0"}`},
		{name: "an empty body", status: http.StatusOK, raw: ` `},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeAPI(t)
			fake.status = tc.status
			fake.raw = tc.raw
			clock := time.Now()
			details, err := fake.lookup(&clock).Latest(context.Background())
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("Latest() error = %v, want ErrUnavailable", err)
			}
			if details != (Details{}) {
				t.Errorf("details = %+v, want an empty answer", details)
			}
		})
	}
}

func TestLatestKeepsTheCacheThroughBadJSON(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v2.1.0", "", "sust-cse-portal-2.1.0.apk")
	clock := time.Now()
	lookup := fake.lookup(&clock)
	good, err := lookup.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	fake.raw = `{"this is not": a list`
	clock = clock.Add(CacheTTL + time.Minute)
	details, err := lookup.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v, want the cached release", err)
	}
	if details != good {
		t.Errorf("details = %+v, want %+v", details, good)
	}
}

func TestLatestRefusesAnUnreachableGitHub(t *testing.T) {
	fake := newFakeAPI(t)
	fake.body = releaseBody("v2.1.0", "", "sust-cse-portal-2.1.0.apk")
	clock := time.Now()
	lookup := fake.lookup(&clock)
	fake.server.Close()
	if _, err := lookup.Latest(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Latest() error = %v, want ErrUnavailable", err)
	}
}

func TestReleasesPagePointsAtTheRepository(t *testing.T) {
	lookup := New("", "")
	want := "https://github.com/raHI-79/250-project/releases/latest"
	if got := lookup.ReleasesPage(); got != want {
		t.Errorf("ReleasesPage() = %q, want %q", got, want)
	}
	other := New("someone/else", "")
	if got := other.ReleasesPage(); got != "https://github.com/someone/else/releases/latest" {
		t.Errorf("ReleasesPage() = %q", got)
	}
}

func TestURLAsksForAPageOfReleases(t *testing.T) {
	lookup := New("", "")
	if got := lookup.URL(); !strings.HasPrefix(got, DefaultAPI+"/repos/raHI-79/250-project/releases?") {
		t.Errorf("URL() = %q", got)
	}
}

func TestReasonNeverCarriesTheToken(t *testing.T) {
	fake := newFakeAPI(t)
	fake.status = http.StatusForbidden
	fake.raw = `{"message":"API rate limit exceeded"}`
	fake.headers["X-RateLimit-Remaining"] = "0"
	clock := time.Now()
	lookup := fake.lookup(&clock).WithToken("super-secret-token")
	_, err := lookup.fetch(context.Background())
	if err == nil {
		t.Fatal("fetch() returned no error, want the refusal")
	}
	text := err.Error()
	if strings.Contains(text, "super-secret-token") {
		t.Fatalf("the error carries the token: %q", text)
	}
	for _, want := range []string{"403", "rate limit exhausted", "API rate limit exceeded"} {
		if !strings.Contains(text, want) {
			t.Errorf("error = %q, want it to mention %q", text, want)
		}
	}
}

func TestVersionCodeOrdersReleases(t *testing.T) {
	cases := []struct {
		latest    string
		installed string
		want      bool
	}{
		{"2.1.0", "2.0.1", true},
		{"v2.1.0", "2.1.0", false},
		{"2.0.1", "2.1.0", false},
		{"3.0.0", "2.9.9", true},
		{"2.1.1", "2.1.0", true},
		{"2.1.0-beta", "2.0.9", true},
	}
	for _, test := range cases {
		if got := Newer(test.latest, test.installed); got != test.want {
			t.Errorf("Newer(%q, %q) = %t, want %t", test.latest, test.installed, got, test.want)
		}
	}
}
