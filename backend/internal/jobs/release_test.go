package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/push"
	"github.com/raHI-79/250-project/backend/internal/release"
)

type fakeReleaseStore struct {
	announced []string
	seen      map[string]bool
	devices   []int64
	browsers  []int64
}

func newReleaseStore(devices, browsers []int64) *fakeReleaseStore {
	return &fakeReleaseStore{seen: map[string]bool{}, devices: devices, browsers: browsers}
}

func (f *fakeReleaseStore) ClaimVersionAnnouncement(_ context.Context, version string) (bool, error) {
	if f.seen[version] {
		return false, nil
	}
	f.seen[version] = true
	f.announced = append(f.announced, version)
	return true, nil
}

func (f *fakeReleaseStore) ListPushTargets(context.Context) ([]int64, error) {
	return f.browsers, nil
}

func (f *fakeReleaseStore) ListDeviceTargets(context.Context) ([]int64, error) {
	return f.devices, nil
}

type fakeReleaseDevices struct {
	title   string
	body    string
	url     string
	tag     string
	reached int
}

func (f *fakeReleaseDevices) Deliver(_ context.Context, userIDs []int64,
	title, body, url, tag string) int {
	f.title, f.body, f.url, f.tag = title, body, url, tag
	f.reached += len(userIDs)
	return len(userIDs)
}

func (f *fakeReleaseDevices) Ready() bool { return true }

type fakeReleasePusher struct {
	payloads []push.Payload
}

func (f *fakeReleasePusher) Queue(_ []int64, payload push.Payload) {
	f.payloads = append(f.payloads, payload)
}

type fakeReleaseSource struct {
	details release.Details
	err     error
	calls   int
}

func (f *fakeReleaseSource) Latest(context.Context) (release.Details, error) {
	f.calls++
	return f.details, f.err
}

func TestReleaseWatcherAnnouncesANewVersionOnce(t *testing.T) {
	silenceLog(t)
	meta := newReleaseStore([]int64{7, 8}, []int64{9})
	devices := &fakeReleaseDevices{}
	pusher := &fakeReleasePusher{}
	source := &fakeReleaseSource{details: release.Details{
		Version: "2.1.0",
		APKURL:  "https://example.test/sust-cse-portal-2.1.0.apk",
	}}
	watcher := NewReleaseWatcher(meta, pusher, devices, source, "https://portal.test")
	sent, version, err := watcher.Announce(context.Background())
	if err != nil {
		t.Fatalf("Announce() error = %v", err)
	}
	if version != "2.1.0" {
		t.Errorf("version = %q, want 2.1.0", version)
	}
	if sent != 3 {
		t.Errorf("sent = %d, want 3", sent)
	}
	if devices.title != "SUST CSE Portal 2.1.0 is available" {
		t.Errorf("title = %q", devices.title)
	}
	if devices.body != ReleaseBody {
		t.Errorf("body = %q, want %q", devices.body, ReleaseBody)
	}
	if devices.url != "https://portal.test" || devices.tag != ReleaseTag {
		t.Errorf("url = %q, tag = %q", devices.url, devices.tag)
	}
	if len(pusher.payloads) != 1 || pusher.payloads[0].Title != devices.title {
		t.Errorf("web push payloads = %#v", pusher.payloads)
	}
	sent, _, err = watcher.Announce(context.Background())
	if err != nil {
		t.Fatalf("second Announce() error = %v", err)
	}
	if sent != 0 {
		t.Errorf("the same version was announced again to %d devices", sent)
	}
	if len(meta.announced) != 1 {
		t.Errorf("announced = %v, want one entry", meta.announced)
	}
}

func TestReleaseWatcherAnnouncesTheNextVersionToo(t *testing.T) {
	silenceLog(t)
	meta := newReleaseStore([]int64{7}, nil)
	devices := &fakeReleaseDevices{}
	source := &fakeReleaseSource{details: release.Details{Version: "2.1.0"}}
	watcher := NewReleaseWatcher(meta, nil, devices, source, "https://portal.test")
	if _, _, err := watcher.Announce(context.Background()); err != nil {
		t.Fatalf("Announce() error = %v", err)
	}
	source.details = release.Details{Version: "2.2.0"}
	sent, version, err := watcher.Announce(context.Background())
	if err != nil {
		t.Fatalf("Announce() error = %v", err)
	}
	if sent != 1 || version != "2.2.0" {
		t.Fatalf("sent = %d, version = %q, want 1 and 2.2.0", sent, version)
	}
	if devices.title != "SUST CSE Portal 2.2.0 is available" {
		t.Errorf("title = %q", devices.title)
	}
}

func TestReleaseWatcherStaysQuietWhenGitHubIsUnreachable(t *testing.T) {
	silenceLog(t)
	meta := newReleaseStore([]int64{7}, nil)
	devices := &fakeReleaseDevices{}
	source := &fakeReleaseSource{err: errors.New("github is down")}
	watcher := NewReleaseWatcher(meta, nil, devices, source, "https://portal.test")
	sent, _, err := watcher.Announce(context.Background())
	if err == nil {
		t.Fatal("Announce() error = nil, want the GitHub failure")
	}
	if sent != 0 || devices.reached != 0 {
		t.Errorf("sent = %d, reached = %d, want nothing delivered", sent, devices.reached)
	}
	if len(meta.announced) != 0 {
		t.Errorf("announced = %v, want nothing", meta.announced)
	}
}

func TestReleaseWatcherIgnoresAReleaseWithoutAVersion(t *testing.T) {
	silenceLog(t)
	meta := newReleaseStore([]int64{7}, nil)
	devices := &fakeReleaseDevices{}
	source := &fakeReleaseSource{details: release.Details{}, err: release.ErrNoAsset}
	watcher := NewReleaseWatcher(meta, nil, devices, source, "https://portal.test")
	sent, _, err := watcher.Announce(context.Background())
	if err == nil {
		t.Fatal("Announce() error = nil, want the missing asset reported")
	}
	if sent != 0 {
		t.Errorf("sent = %d, want 0", sent)
	}
}
