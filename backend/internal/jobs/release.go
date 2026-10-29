package jobs

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/push"
)

const ReleaseTitle = "SUST CSE Portal"

type ReleaseStore interface {
	ClaimVersionAnnouncement(ctx context.Context, version string) (bool, error)
	ListPushTargets(ctx context.Context) ([]int64, error)
}

type ReleasePusher interface {
	Queue(userIDs []int64, payload push.Payload)
}

func ReleaseBody(version string) string {
	return "SUST CSE Portal updated to " + version + " — open the app to get the latest"
}

func AnnounceRelease(ctx context.Context, meta ReleaseStore, pusher ReleasePusher,
	version, origin string) (int, error) {
	if pusher == nil || version == "" {
		return 0, nil
	}
	fresh, err := meta.ClaimVersionAnnouncement(ctx, version)
	if err != nil || !fresh {
		return 0, err
	}
	targets, err := meta.ListPushTargets(ctx)
	if err != nil {
		return 0, err
	}
	if len(targets) == 0 {
		return 0, nil
	}
	pusher.Queue(targets, push.Payload{
		Title: ReleaseTitle,
		Body:  ReleaseBody(version),
		URL:   origin,
		Tag:   "release",
	})
	return len(targets), nil
}
