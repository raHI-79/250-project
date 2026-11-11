package handler

import (
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/store"
)

func hasAlert(alerts []alertItem, key string) bool {
	for _, alert := range alerts {
		if alert.Key == key {
			return true
		}
	}
	return false
}

func busyPeriod(failures int) store.RequestVolume {
	return store.RequestVolume{Requests: minRequestVolume, Failures: failures}
}

func TestAlertRulesFireOnTheRightNumbers(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-time.Hour)
	stale := now.Add(-48 * time.Hour)
	tests := []struct {
		name      string
		health    store.SystemTotals
		requests  store.RequestVolume
		push      store.PushVolume
		backup    store.BackupState
		spike     int
		deletions int
		wants     []string
		missing   []string
	}{
		{
			name:     "a healthy portal only misses its backup",
			health:   store.SystemTotals{EmailsSent: 20},
			requests: busyPeriod(0),
			push:     store.PushVolume{Sent: 40},
			backup:   store.BackupState{LastSuccessAt: &fresh},
			missing: []string{AlertServerErrors, AlertEmailFailure, AlertPushFailure,
				AlertBackupStale, AlertSignInSpike, AlertSheetDeletes},
		},
		{
			name:     "a run of server errors in a busy period is reported",
			requests: busyPeriod(serverErrorSpike),
			backup:   store.BackupState{LastSuccessAt: &fresh},
			wants:    []string{AlertServerErrors},
		},
		{
			name:     "the same run of server errors in a quiet period stays quiet",
			requests: store.RequestVolume{Requests: minRequestVolume - 1, Failures: serverErrorSpike},
			backup:   store.BackupState{LastSuccessAt: &fresh},
			missing:  []string{AlertServerErrors},
		},
		{
			name:     "one server error in a busy period stays quiet",
			requests: busyPeriod(1),
			backup:   store.BackupState{LastSuccessAt: &fresh},
			missing:  []string{AlertServerErrors},
		},
		{
			name:     "an email failure share above a fifth is reported",
			health:   store.SystemTotals{EmailsSent: 5, EmailsFailed: 5},
			requests: busyPeriod(0),
			backup:   store.BackupState{LastSuccessAt: &fresh},
			wants:    []string{AlertEmailFailure},
		},
		{
			name:     "a tiny number of email failures stays quiet",
			health:   store.SystemTotals{EmailsSent: 2, EmailsFailed: 1},
			requests: busyPeriod(0),
			backup:   store.BackupState{LastSuccessAt: &fresh},
			missing:  []string{AlertEmailFailure},
		},
		{
			name:     "a notification failure share above a fifth is reported",
			requests: busyPeriod(0),
			push:     store.PushVolume{Sent: 10, Failed: 15},
			backup:   store.BackupState{LastSuccessAt: &fresh},
			wants:    []string{AlertPushFailure},
		},
		{
			name:     "a handful of notification failures stays quiet",
			requests: busyPeriod(0),
			push:     store.PushVolume{Sent: 2, Failed: 3},
			backup:   store.BackupState{LastSuccessAt: &fresh},
			missing:  []string{AlertPushFailure},
		},
		{
			name:     "failures the server already cleaned up stay quiet",
			requests: busyPeriod(0),
			push:     store.PushVolume{Sent: 10, Failed: 15, Cleaned: 15},
			backup:   store.BackupState{LastSuccessAt: &fresh},
			missing:  []string{AlertPushFailure},
		},
		{
			name:     "the failures beyond the cleaned ones still count",
			requests: busyPeriod(0),
			push:     store.PushVolume{Sent: 10, Failed: 15, Cleaned: 5},
			backup:   store.BackupState{LastSuccessAt: &fresh},
			wants:    []string{AlertPushFailure},
		},
		{
			name:     "a backup older than 36 hours is reported",
			requests: busyPeriod(0),
			backup:   store.BackupState{LastSuccessAt: &stale},
			wants:    []string{AlertBackupStale},
		},
		{
			name:     "a missing backup is reported",
			requests: busyPeriod(0),
			backup:   store.BackupState{},
			wants:    []string{AlertBackupStale},
		},
		{
			name:     "a failed sign-in spike from one address is reported",
			requests: busyPeriod(0),
			backup:   store.BackupState{LastSuccessAt: &fresh},
			spike:    signInSpikeLimit,
			wants:    []string{AlertSignInSpike},
		},
		{
			name:      "a deleted attendance sheet is reported",
			requests:  busyPeriod(0),
			backup:    store.BackupState{LastSuccessAt: &fresh},
			deletions: 1,
			wants:     []string{AlertSheetDeletes},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			alerts := alertsFor(alertFacts{
				Health:    tc.health,
				Requests:  tc.requests,
				Push:      tc.push,
				Backup:    tc.backup,
				Spike:     tc.spike,
				Deletions: tc.deletions,
				Now:       now,
			})
			for _, key := range tc.wants {
				if !hasAlert(alerts, key) {
					t.Errorf("the %s alert is missing from %+v", key, alerts)
				}
			}
			for _, key := range tc.missing {
				if hasAlert(alerts, key) {
					t.Errorf("the %s alert fired although it should stay quiet", key)
				}
			}
		})
	}
}

func TestADismissedAlertStaysQuietForADay(t *testing.T) {
	now := time.Now()
	facts := alertFacts{
		Requests: busyPeriod(serverErrorSpike),
		Push:     store.PushVolume{Sent: 10, Failed: 15},
		Backup:   store.BackupState{},
		Now:      now,
	}
	loud := alertsFor(facts)
	for _, key := range []string{AlertServerErrors, AlertPushFailure, AlertBackupStale} {
		if !hasAlert(loud, key) {
			t.Fatalf("the %s alert did not fire before the dismissal", key)
		}
	}
	facts.Quiet = map[string]time.Time{
		AlertServerErrors: now.Add(store.AlertQuietWindow),
		AlertPushFailure:  now.Add(time.Minute),
	}
	quiet := alertsFor(facts)
	if hasAlert(quiet, AlertServerErrors) || hasAlert(quiet, AlertPushFailure) {
		t.Errorf("a dismissed alert came back: %+v", quiet)
	}
	if !hasAlert(quiet, AlertBackupStale) {
		t.Errorf("dismissing one alert silenced another: %+v", quiet)
	}
	facts.Quiet = map[string]time.Time{AlertServerErrors: now.Add(-time.Minute)}
	back := alertsFor(facts)
	if !hasAlert(back, AlertServerErrors) {
		t.Errorf("the alert stayed quiet after its window ran out: %+v", back)
	}
}

func TestEveryAlertCarriesAMessageAToneAndASection(t *testing.T) {
	alerts := alertsFor(alertFacts{
		Health:    store.SystemTotals{EmailsSent: 1, EmailsFailed: 9},
		Requests:  busyPeriod(99),
		Push:      store.PushVolume{Sent: 5, Failed: 40},
		Backup:    store.BackupState{},
		Spike:     99,
		Deletions: 3,
		Now:       time.Now(),
	})
	if len(alerts) != 6 {
		t.Fatalf("alertsFor() reported %d alerts on a broken portal, want all six", len(alerts))
	}
	for _, alert := range alerts {
		if alert.Message == "" || alert.Tone == "" {
			t.Errorf("alert %+v has no message or tone", alert)
		}
		if alert.Section == "" {
			t.Errorf("alert %+v has no System health section to link to", alert)
		}
		if !KnownAlert(alert.Key) {
			t.Errorf("the alert %q cannot be dismissed because it is not a known key", alert.Key)
		}
	}
}

func TestTheFailureShareLeavesOutWhatWasCleanedUp(t *testing.T) {
	tests := []struct {
		name   string
		volume store.PushVolume
		want   float64
	}{
		{name: "nothing sent", volume: store.PushVolume{}, want: 0},
		{name: "all delivered", volume: store.PushVolume{Sent: 10}, want: 0},
		{name: "half failed", volume: store.PushVolume{Sent: 10, Failed: 10}, want: 0.5},
		{
			name:   "every failure was an expired subscription",
			volume: store.PushVolume{Sent: 10, Failed: 10, Cleaned: 10},
			want:   0,
		},
		{
			name:   "one real failure among the expired ones",
			volume: store.PushVolume{Sent: 8, Failed: 10, Cleaned: 9},
			want:   1.0 / 9.0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.volume.Share()
			if diff := got - test.want; diff > 0.0001 || diff < -0.0001 {
				t.Errorf("Share() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestThePushReasonLabelsReadPlainly(t *testing.T) {
	cases := map[string]string{
		store.PushReasonExpired:     "Web push · Expired or unregistered",
		store.PushReasonInvalidKey:  "Web push · Invalid key or refused",
		store.PushReasonRateLimited: "Web push · Rate limited",
		store.PushReasonNetwork:     "Web push · Could not be reached",
		store.PushReasonPayload:     "Web push · Payload refused",
		store.PushReasonOther:       "Web push · Something else",
	}
	for reason, want := range cases {
		if got := pushReasonLabel(store.ChannelWebPush, reason); got != want {
			t.Errorf("pushReasonLabel(web, %s) = %q, want %q", reason, got, want)
		}
	}
	if got := pushReasonLabel(store.ChannelFCM, store.PushReasonExpired); got !=
		"App notifications · Expired or unregistered" {
		t.Errorf("pushReasonLabel(fcm, expired) = %q", got)
	}
	if got := pushReasonLabel("something", "new"); got != "something · new" {
		t.Errorf("pushReasonLabel() made up a label for an unknown pair: %q", got)
	}
}

func TestOnlyKnownAlertsCanBeDismissed(t *testing.T) {
	for _, key := range []string{AlertServerErrors, AlertEmailFailure, AlertPushFailure,
		AlertBackupStale, AlertSignInSpike, AlertSheetDeletes} {
		if !KnownAlert(key) {
			t.Errorf("KnownAlert(%q) = false, want true", key)
		}
	}
	for _, key := range []string{"", "something_else", "server_errors ", "SERVER_ERRORS"} {
		if KnownAlert(key) {
			t.Errorf("KnownAlert(%q) = true, want false", key)
		}
	}
}
