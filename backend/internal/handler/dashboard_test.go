package handler

import (
	"github.com/raHI-79/250-project/backend/internal/store"
	"testing"
	"time"
)

func hasAlert(alerts []alertItem, key string) bool {
	for _, alert := range alerts {
		if alert.Key == key {
			return true
		}
	}
	return false
}

func TestAlertRulesFireOnTheRightNumbers(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-time.Hour)
	stale := now.Add(-48 * time.Hour)
	tests := []struct {
		name      string
		health    store.SystemTotals
		backup    store.BackupState
		spike     int
		deletions int
		wants     []string
		missing   []string
	}{
		{
			name:    "a healthy portal only misses its backup",
			health:  store.SystemTotals{EmailsSent: 20, PushSent: 20, FCMSent: 20},
			backup:  store.BackupState{LastSuccessAt: &fresh},
			missing: []string{AlertServerErrors, AlertEmailFailure, AlertPushFailure, AlertBackupStale, AlertSignInSpike, AlertSheetDeletes},
		},
		{
			name:   "a 5xx spike is reported",
			health: store.SystemTotals{Server5xx: serverErrorSpike},
			backup: store.BackupState{LastSuccessAt: &fresh},
			wants:  []string{AlertServerErrors},
		},
		{
			name:   "an email failure share above a fifth is reported",
			health: store.SystemTotals{EmailsSent: 5, EmailsFailed: 5},
			backup: store.BackupState{LastSuccessAt: &fresh},
			wants:  []string{AlertEmailFailure},
		},
		{
			name:    "a tiny number of email failures stays quiet",
			health:  store.SystemTotals{EmailsSent: 2, EmailsFailed: 1},
			backup:  store.BackupState{LastSuccessAt: &fresh},
			missing: []string{AlertEmailFailure},
		},
		{
			name:   "a push failure share above a fifth is reported",
			health: store.SystemTotals{FCMSent: 5, FCMFailed: 5},
			backup: store.BackupState{LastSuccessAt: &fresh},
			wants:  []string{AlertPushFailure},
		},
		{
			name:   "a backup older than 36 hours is reported",
			backup: store.BackupState{LastSuccessAt: &stale},
			wants:  []string{AlertBackupStale},
		},
		{
			name:   "a missing backup is reported",
			backup: store.BackupState{},
			wants:  []string{AlertBackupStale},
		},
		{
			name:   "a failed sign-in spike from one address is reported",
			backup: store.BackupState{LastSuccessAt: &fresh},
			spike:  signInSpikeLimit,
			wants:  []string{AlertSignInSpike},
		},
		{
			name:      "a deleted attendance sheet is reported",
			backup:    store.BackupState{LastSuccessAt: &fresh},
			deletions: 1,
			wants:     []string{AlertSheetDeletes},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			alerts := alertsFor(tc.health, tc.backup, tc.spike, tc.deletions, now)
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

func TestEveryAlertCarriesAMessageAndATone(t *testing.T) {
	alerts := alertsFor(store.SystemTotals{Server5xx: 99, EmailsSent: 1, EmailsFailed: 9},
		store.BackupState{}, 99, 3, time.Now())
	if len(alerts) == 0 {
		t.Fatal("alertsFor() reported nothing on a broken portal")
	}
	for _, alert := range alerts {
		if alert.Message == "" || alert.Tone == "" {
			t.Errorf("alert %+v has no message or tone", alert)
		}
	}
}
