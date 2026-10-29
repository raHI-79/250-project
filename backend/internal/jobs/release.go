package jobs

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/push"
)

const (
	ReleaseTitle = "SUST CSE Portal"
	ReleaseTag   = "release"
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

func ReleaseBody(version string) string {
	return "SUST CSE Portal updated to " + version + " — open the app to get the latest"
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
	reached := 0
	if pusher != nil {
		targets, err := meta.ListPushTargets(ctx)
		if err != nil {
			return 0, err
		}
		if len(targets) > 0 {
			pusher.Queue(targets, push.Payload{
				Title: ReleaseTitle,
				Body:  ReleaseBody(version),
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
		reached += devices.Deliver(ctx, targets, ReleaseTitle, ReleaseBody(version), origin, ReleaseTag)
	}
	return reached, nil
}
