package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/validate"
)

func TestCRClassRequestNote(t *testing.T) {
	silenceLog(t)
	tomorrow, _, _ := crDates(t)
	base := crRequestBody(tomorrow)
	withNote := func(fields string) string {
		return strings.Replace(base, `"room":"1003"`, `"room":"1003",`+fields, 1)
	}
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantError    string
		wantNote     string
		wantAudience string
	}{
		{name: "no note at all", body: base, wantStatus: http.StatusCreated},
		{
			name:         "a note for the teacher",
			body:         withNote(`"note":"The projector is broken","note_audience":"teacher"`),
			wantStatus:   http.StatusCreated,
			wantNote:     "The projector is broken",
			wantAudience: validate.NoteTeacher,
		},
		{
			name:         "a note for the students",
			body:         withNote(`"note":"Bring the lab sheet","note_audience":"students"`),
			wantStatus:   http.StatusCreated,
			wantNote:     "Bring the lab sheet",
			wantAudience: validate.NoteStudents,
		},
		{
			name:         "a note for both",
			body:         withNote(`"note":"Room changed","note_audience":"both"`),
			wantStatus:   http.StatusCreated,
			wantNote:     "Room changed",
			wantAudience: validate.NoteBoth,
		},
		{
			name:       "a note without an audience",
			body:       withNote(`"note":"Who should read this?"`),
			wantStatus: http.StatusBadRequest,
			wantError:  "Note audience is required.",
		},
		{
			name:       "a broken audience",
			body:       withNote(`"note":"Hello","note_audience":"everyone"`),
			wantStatus: http.StatusBadRequest,
			wantError:  "Note audience must be one of teacher, students, both.",
		},
		{
			name:       "a note over the limit",
			body:       withNote(`"note":"` + strings.Repeat("a", validate.MaxNote+1) + `","note_audience":"both"`),
			wantStatus: http.StatusBadRequest,
			wantError:  "Note must be at most 500 characters long.",
		},
		{
			name:       "an audience without a note is ignored",
			body:       withNote(`"note_audience":"both"`),
			wantStatus: http.StatusCreated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			classes := newCRStore()
			handler := newCRHandler(t, classes, &fakeNotifier{})
			rec := crCall(t, handler.CreateClassRequest(), http.MethodPost, "/api/cr/class-requests", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if len(classes.events) != 1 {
				t.Fatalf("the store received %d events, want 1", len(classes.events))
			}
			saved := classes.events[0]
			if saved.Note != tc.wantNote {
				t.Errorf("stored note = %q, want %q", saved.Note, tc.wantNote)
			}
			if saved.NoteAudience != tc.wantAudience {
				t.Errorf("stored audience = %q, want %q", saved.NoteAudience, tc.wantAudience)
			}
		})
	}
}
