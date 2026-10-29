package jobs

import (
	"context"
	"log"
	"time"

	"github.com/raHI-79/250-project/backend/internal/push"
	"github.com/raHI-79/250-project/backend/internal/release"
)

const (
	ReleaseTag      = "release"
	ReleaseBody     = "Tap to update"
	ReleaseInterval = 15 * time.Minute
	releaseSuffix   = " is available"
)

type ReleaseStore interface {
	ClaimVersionAnnouncement(ctx context.Context, version string) (bool, error)
	ListPushTargets(ctx context.Context) ([]int64, error)
	ListDeviceTargets(ctx context.Context) ([]int64, error)
}

type ReleaseDevices interface {
	Deliver(ctx context.Context, userIDs []int64, title, body, url, tag string) int
	Ready() bool
}

type ReleasePusher interface {
	Queue(userIDs []int64, payload push.Payload)
}

type ReleaseSource interface {
	Latest(ctx context.Context) (release.Details, error)
}

func ReleaseTitle(version string) string {
	if version == "" {
		return "SUST CSE Portal" + releaseSuffix
	}
	return "SUST CSE Portal " + version + releaseSuffix
}

func AnnounceRelease(ctx context.Context, meta ReleaseStore, pusher ReleasePusher,
	devices ReleaseDevices, version, origin string) (int, error) {
	ready := devices != nil && devices.Ready()
	if version == "" || (pusher == nil && !ready) {
		return 0, nil
	}
	fresh, err := meta.ClaimVersionAnnouncement(ctx, version)
	if err != nil || !fresh {
		return 0, err
	}
	title := ReleaseTitle(version)
	reached := 0
	if pusher != nil {
		targets, err := meta.ListPushTargets(ctx)
		if err != nil {
			return 0, err
		}
		if len(targets) > 0 {
			pusher.Queue(targets, push.Payload{
				Title: title,
				Body:  ReleaseBody,
				URL:   origin,
				Tag:   ReleaseTag,
			})
			reached += len(targets)
		}
	}
	if ready {
		targets, err := meta.ListDeviceTargets(ctx)
		if err != nil {
			return reached, err
		}
		reached += devices.Deliver(ctx, targets, title, ReleaseBody, origin, ReleaseTag)
	}
	return reached, nil
}

type ReleaseWatcher struct {
	meta    ReleaseStore
	pusher  ReleasePusher
	devices ReleaseDevices
	source  ReleaseSource
	origin  string
}

func NewReleaseWatcher(meta ReleaseStore, pusher ReleasePusher, devices ReleaseDevices,
	source ReleaseSource, origin string) *ReleaseWatcher {
	return &ReleaseWatcher{meta: meta, pusher: pusher, devices: devices, source: source, origin: origin}
}

func (w *ReleaseWatcher) Run(ctx context.Context, interval time.Duration) {
	w.RunOnce(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.RunOnce(ctx)
		}
	}
}

func (w *ReleaseWatcher) RunOnce(ctx context.Context) {
	sent, version, err := w.Announce(ctx)
	if err != nil {
		log.Printf("jobs: could not announce the release: %v", err)
		return
	}
	if sent > 0 {
		log.Printf("jobs: announced version %s to %d devices", version, sent)
	}
}

func (w *ReleaseWatcher) Announce(ctx context.Context) (int, string, error) {
	if w.source == nil {
		return 0, "", nil
	}
	details, err := w.source.Latest(ctx)
	if err != nil && details.Version == "" {
		return 0, "", err
	}
	sent, err := AnnounceRelease(ctx, w.meta, w.pusher, w.devices, details.Version, w.origin)
	return sent, details.Version, err
}
