package jobs

import (
	"context"
)

type Announcer struct {
	meta    ReleaseStore
	pusher  ReleasePusher
	devices ReleaseDevices
	origin  string
}

func NewAnnouncer(meta ReleaseStore, pusher ReleasePusher, devices ReleaseDevices,
	origin string) *Announcer {
	return &Announcer{meta: meta, pusher: pusher, devices: devices, origin: origin}
}

func (a *Announcer) Announce(ctx context.Context, version string) (int, error) {
	if a == nil || version == "" {
		return 0, nil
	}
	if err := a.meta.ForgetVersionAnnouncement(ctx, version); err != nil {
		return 0, err
	}
	return AnnounceRelease(ctx, a.meta, a.pusher, a.devices, version, a.origin)
}
