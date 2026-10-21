package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"strings"
	"testing"
	"time"
)

var profileNow = time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC)

type profileUpdateCall struct {
	userID int64
	role   string
	update store.ProfileUpdate
}

type fakeProfileStore struct {
	profiles    map[int64]store.Profile
	people      map[int64]store.PersonCard
	hashes      map[int64]string
	allowedIDs  map[int64]bool
	updates     []profileUpdateCall
	passwords   []string
	oldPhoto    string
	getErr      error
	updateErr   error
	passwordErr error
}

func newProfileStore(t *testing.T) *fakeProfileStore {
	t.Helper()
	return &fakeProfileStore{
		profiles: map[int64]store.Profile{
			1: {
				UserID:      1,
				Role:        "student",
				Status:      store.StatusActive,
				Identifier:  "2023331001",
				PhotoPath:   "/uploads/photos/aaaa.png",
				CreatedAt:   profileNow,
				FullName:    "Active Student",
				RegNo:       "2023331001",
				Batch:       "CSE-23",
				Section:     "A",
				Email:       "2023331001@student.sust.edu",
				Phone:       "01712345678",
				BloodGroup:  "O+",
				DateOfBirth: "2003-05-14",
				DropCourses: []store.Course{{ID: 4, Code: "CSE-233", Title: "Algorithms"}},
			},
			2: {
				UserID:      2,
				Role:        "student",
				Status:      store.StatusActive,
				Identifier:  "2023331002",
				CreatedAt:   profileNow,
				FullName:    "Class Representative",
				RegNo:       "2023331002",
				Batch:       "CSE-23",
				Section:     "B",
				Email:       "2023331002@student.sust.edu",
				Phone:       "01712345679",
				BloodGroup:  "A+",
				DateOfBirth: "2003-06-20",
				IsCR:        true,
				DropCourses: []store.Course{},
			},
			6: {
				UserID:      6,
				Role:        "teacher",
				Status:      store.StatusActive,
				Identifier:  "T-001",
				PhotoPath:   "/uploads/photos/bbbb.jpg",
				CreatedAt:   profileNow,
				FullName:    "Active Teacher",
				TeacherID:   "T-001",
				Designation: "Assistant Professor",
				Email:       "teacher@sust.edu",
				Phone:       "01812345678",
				BloodGroup:  "B+",
				DateOfBirth: "1988-03-09",
				OfficeRoom:  "AB-401",
				JoiningDate: "2015-01-15",
				Bio:         "Teaches algorithms.",
				DropCourses: []store.Course{},
			},
			7: {
				UserID:       7,
				Role:         "admin",
				Status:       store.StatusActive,
				Identifier:   "admin",
				CreatedAt:    profileNow,
				ContactEmail: "admin@sust.edu",
				ContactPhone: "01912345678",
				DropCourses:  []store.Course{},
			},
		},
		people: map[int64]store.PersonCard{
			1: {
				UserID: 1, Role: "student", Name: "Active Student", Batch: "CSE-23", Section: "A",
				Email: "2023331001@student.sust.edu", Phone: "01712345678", BloodGroup: "O+",
				PhotoPath: "/uploads/photos/aaaa.png",
			},
			2: {
				UserID: 2, Role: "cr", Name: "Class Representative", Batch: "CSE-23", Section: "B",
				Email: "2023331002@student.sust.edu", Phone: "01712345679", BloodGroup: "A+",
			},
			6: {
				UserID: 6, Role: "teacher", Name: "Active Teacher", Designation: "Assistant Professor",
				OfficeRoom: "AB-401", Email: "teacher@sust.edu", Phone: "01812345678", BloodGroup: "B+",
			},
		},
		hashes:     map[int64]string{1: cachedHash(t, studentPassword)},
		allowedIDs: map[int64]bool{4: true, 9: true},
	}
}

func (f *fakeProfileStore) GetProfile(ctx context.Context, userID int64) (store.Profile, error) {
	if f.getErr != nil {
		return store.Profile{}, f.getErr
	}
	profile, ok := f.profiles[userID]
	if !ok {
		return store.Profile{}, store.ErrNotFound
	}
	return profile, nil
}

func (f *fakeProfileStore) UpdateProfile(ctx context.Context, userID int64, role string, update store.ProfileUpdate) (string, error) {
	if f.updateErr != nil {
		return "", f.updateErr
	}
	f.updates = append(f.updates, profileUpdateCall{userID: userID, role: role, update: update})
	return f.oldPhoto, nil
}

func (f *fakeProfileStore) GetPasswordHash(ctx context.Context, userID int64) (string, error) {
	hash, ok := f.hashes[userID]
	if !ok {
		return "", store.ErrNotFound
	}
	return hash, nil
}

func (f *fakeProfileStore) SetPassword(ctx context.Context, userID int64, passwordHash string) (int, error) {
	if f.passwordErr != nil {
		return 0, f.passwordErr
	}
	f.passwords = append(f.passwords, passwordHash)
	return 1, nil
}

func (f *fakeProfileStore) FilterDropCourseIDs(ctx context.Context, courseIDs []int64) ([]int64, error) {
	allowed := make([]int64, 0, len(courseIDs))
	for _, courseID := range courseIDs {
		if f.allowedIDs[courseID] {
			allowed = append(allowed, courseID)
		}
	}
	return allowed, nil
}

func (f *fakeProfileStore) FindPerson(ctx context.Context, userID int64) (store.PersonCard, error) {
	card, ok := f.people[userID]
	if !ok {
		return store.PersonCard{}, store.ErrNotFound
	}
	return card, nil
}

func newProfileHandler(t *testing.T, profiles ProfileStore, photos PhotoStore) *Profile {
	t.Helper()
	return NewProfile(accountConfig(t, "test"), profiles, photos)
}

func TestProfileGet(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		userID      int64
		role        string
		wantStatus  int
		wantFields  map[string]any
		wantMissing []string
	}{
		{
			name:       "student",
			userID:     1,
			role:       auth.RoleStudent,
			wantStatus: http.StatusOK,
			wantFields: map[string]any{
				"user_id":       float64(1),
				"role":          auth.RoleStudent,
				"full_name":     "Active Student",
				"reg_no":        "2023331001",
				"batch":         "CSE-23",
				"section":       "A",
				"email":         "2023331001@student.sust.edu",
				"phone":         "01712345678",
				"blood_group":   "O+",
				"date_of_birth": "2003-05-14",
				"is_cr":         false,
				"photo_url":     "/uploads/photos/aaaa.png",
			},
			wantMissing: []string{"designation", "contact_email"},
		},
		{
			name:       "class representative",
			userID:     2,
			role:       auth.RoleCR,
			wantStatus: http.StatusOK,
			wantFields: map[string]any{"role": auth.RoleCR, "is_cr": true, "section": "B"},
		},
		{
			name:       "teacher",
			userID:     6,
			role:       auth.RoleTeacher,
			wantStatus: http.StatusOK,
			wantFields: map[string]any{
				"role":          auth.RoleTeacher,
				"name":          "Active Teacher",
				"identifier":    "T-001",
				"designation":   "Assistant Professor",
				"email":         "teacher@sust.edu",
				"phone":         "01812345678",
				"blood_group":   "B+",
				"date_of_birth": "1988-03-09",
				"office_room":   "AB-401",
				"joining_date":  "2015-01-15",
				"bio":           "Teaches algorithms.",
			},
			wantMissing: []string{"reg_no", "batch", "drop_courses", "contact_phone"},
		},
		{
			name:       "admin",
			userID:     7,
			role:       auth.RoleAdmin,
			wantStatus: http.StatusOK,
			wantFields: map[string]any{
				"role":          auth.RoleAdmin,
				"identifier":    "admin",
				"contact_email": "admin@sust.edu",
				"contact_phone": "01912345678",
			},
			wantMissing: []string{"reg_no", "phone", "email"},
		},
		{name: "unknown account", userID: 99, role: auth.RoleStudent, wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles := newProfileStore(t)
			handler := newProfileHandler(t, profiles, testPhotos(t))
			request := jsonRequest(t, http.MethodGet, "/api/profile", "", test.userID, test.role)
			rec := serve(handler.Get(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			payload := decodeBody(t, rec)
			for name, want := range test.wantFields {
				if got := payload[name]; got != want {
					t.Errorf("%s = %v, want %v", name, got, want)
				}
			}
			for _, name := range test.wantMissing {
				if _, present := payload[name]; present {
					t.Errorf("the %s profile carries the %q field", test.role, name)
				}
			}
			if _, present := payload["password_hash"]; present {
				t.Error("the profile carries a password hash")
			}
			if test.role == auth.RoleStudent || test.role == auth.RoleCR {
				if _, present := payload["drop_courses"]; test.userID == 1 && !present {
					t.Error("the student profile hides the drop courses")
				}
			}
		})
	}
}

func TestProfilePatchRules(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		role       string
		userID     int64
		values     map[string][]string
		withPhoto  bool
		wantStatus int
		wantError  string
		check      func(t *testing.T, update store.ProfileUpdate)
	}{
		{
			name:       "student changes the phone",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"phone": {"01799999999"}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, update store.ProfileUpdate) {
				if update.Phone == nil || *update.Phone != "01799999999" {
					t.Errorf("phone = %v, want 01799999999", update.Phone)
				}
			},
		},
		{
			name:       "student replaces the drop courses",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"drop_course_ids": {"4", "9"}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, update store.ProfileUpdate) {
				if update.DropCourseIDs == nil {
					t.Fatal("the update carries no drop courses")
				}
				if len(*update.DropCourseIDs) != 2 {
					t.Errorf("drop courses = %v, want two of them", *update.DropCourseIDs)
				}
			},
		},
		{
			name:       "student clears the drop courses",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"drop_course_ids": {""}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, update store.ProfileUpdate) {
				if update.DropCourseIDs == nil {
					t.Fatal("the update carries no drop course set")
				}
				if len(*update.DropCourseIDs) != 0 {
					t.Errorf("drop courses = %v, want an empty set", *update.DropCourseIDs)
				}
			},
		},
		{
			name:       "student uploads a photo",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{},
			withPhoto:  true,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, update store.ProfileUpdate) {
				if update.PhotoPath == nil || *update.PhotoPath == "" {
					t.Error("the update carries no photo")
				}
			},
		},
		{
			name:       "student cannot change the batch",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"batch": {"CSE-22"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
		{
			name:       "student cannot change the email",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"email": {"other@student.sust.edu"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
		{
			name:       "student cannot change the name",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"full_name": {"Someone Else"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
		{
			name:       "student cannot set an office room",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"office_room": {"AB-401"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
		{
			name:       "bad phone",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"phone": {"+8801712345678"}},
			wantStatus: http.StatusBadRequest,
			wantError:  validate.MessagePhone,
		},
		{
			name:       "unknown drop course",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{"drop_course_ids": {"77"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageUnknownDropCourse,
		},
		{
			name:       "nothing to change",
			role:       auth.RoleStudent,
			userID:     1,
			values:     map[string][]string{},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageNothingToUpdate,
		},
		{
			name:       "teacher changes the office room and bio",
			role:       auth.RoleTeacher,
			userID:     6,
			values:     map[string][]string{"office_room": {"AB-502"}, "bio": {"Teaches databases."}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, update store.ProfileUpdate) {
				if update.OfficeRoom == nil || *update.OfficeRoom != "AB-502" {
					t.Errorf("office_room = %v, want AB-502", update.OfficeRoom)
				}
				if update.Bio == nil || *update.Bio != "Teaches databases." {
					t.Errorf("bio = %v, want the new bio", update.Bio)
				}
			},
		},
		{
			name:       "teacher bio over the limit",
			role:       auth.RoleTeacher,
			userID:     6,
			values:     map[string][]string{"bio": {strings.Repeat("a", MaxBioLength+1)}},
			wantStatus: http.StatusBadRequest,
			wantError:  "Bio must be at most 500 characters long.",
		},
		{
			name:       "teacher cannot change the designation",
			role:       auth.RoleTeacher,
			userID:     6,
			values:     map[string][]string{"designation": {"Professor"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
		{
			name:       "teacher cannot change the teacher id",
			role:       auth.RoleTeacher,
			userID:     6,
			values:     map[string][]string{"teacher_id": {"T-999"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
		{
			name:       "teacher cannot set drop courses",
			role:       auth.RoleTeacher,
			userID:     6,
			values:     map[string][]string{"drop_course_ids": {"4"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
		{
			name:       "admin changes the contact details",
			role:       auth.RoleAdmin,
			userID:     7,
			values:     map[string][]string{"contact_email": {"office@sust.edu"}, "contact_phone": {"01888888888"}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, update store.ProfileUpdate) {
				if update.ContactEmail == nil || *update.ContactEmail != "office@sust.edu" {
					t.Errorf("contact_email = %v, want office@sust.edu", update.ContactEmail)
				}
				if update.ContactPhone == nil || *update.ContactPhone != "01888888888" {
					t.Errorf("contact_phone = %v, want 01888888888", update.ContactPhone)
				}
			},
		},
		{
			name:       "admin cannot change a phone field of its own",
			role:       auth.RoleAdmin,
			userID:     7,
			values:     map[string][]string{"phone": {"01888888888"}},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageProfileField,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles := newProfileStore(t)
			photos := testPhotos(t)
			handler := newProfileHandler(t, profiles, photos)
			files := []formFile{}
			if test.withPhoto {
				files = append(files, formFile{field: "photo", name: "me.png", data: pngPhoto(t, 240, 240)})
			}
			request := multipartRequest(t, http.MethodPatch, "/api/profile", test.values, files...)
			request = withCurrentUser(request, test.userID, test.role)
			rec := serve(handler.Patch(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(profiles.updates) != 0 {
					t.Error("the store received an update although the request was refused")
				}
				return
			}
			if len(profiles.updates) != 1 {
				t.Fatalf("the store received %d updates, want 1", len(profiles.updates))
			}
			call := profiles.updates[0]
			if call.userID != test.userID || call.role != test.role {
				t.Errorf("update was made for %d as %q, want %d as %q", call.userID, call.role, test.userID, test.role)
			}
			if test.check != nil {
				test.check(t, call.update)
			}
		})
	}
}

func TestProfilePatchReplacesTheOldPhoto(t *testing.T) {
	silenceLog(t)
	profiles := newProfileStore(t)
	photos := testPhotos(t)
	handler := newProfileHandler(t, profiles, photos)
	first := multipartRequest(t, http.MethodPatch, "/api/profile", map[string][]string{},
		formFile{field: "photo", name: "me.png", data: pngPhoto(t, 240, 240)})
	first = withCurrentUser(first, 1, auth.RoleStudent)
	if rec := serve(handler.Patch(), first); rec.Code != http.StatusOK {
		t.Fatalf("first patch status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	stored := *profiles.updates[0].update.PhotoPath
	profiles.oldPhoto = stored
	second := multipartRequest(t, http.MethodPatch, "/api/profile", map[string][]string{},
		formFile{field: "photo", name: "me.png", data: pngPhoto(t, 260, 260)})
	second = withCurrentUser(second, 1, auth.RoleStudent)
	if rec := serve(handler.Patch(), second); rec.Code != http.StatusOK {
		t.Fatalf("second patch status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if savedPhotos(t, photos) != 1 {
		t.Errorf("%d photos are on disk, want only the newest one", savedPhotos(t, photos))
	}
}

func TestProfilePatchRemovesThePhotoWhenTheStoreFails(t *testing.T) {
	silenceLog(t)
	profiles := newProfileStore(t)
	profiles.updateErr = errors.New("database is down")
	photos := testPhotos(t)
	handler := newProfileHandler(t, profiles, photos)
	request := multipartRequest(t, http.MethodPatch, "/api/profile", map[string][]string{"phone": {"01799999999"}},
		formFile{field: "photo", name: "me.png", data: pngPhoto(t, 240, 240)})
	request = withCurrentUser(request, 1, auth.RoleStudent)
	rec := serve(handler.Patch(), request)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body = %q)", rec.Code, rec.Body.String())
	}
	if savedPhotos(t, photos) != 0 {
		t.Error("the photo stayed on disk although the update failed")
	}
}

func TestProfileChangePassword(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		userID     int64
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{
			name:       "changed",
			userID:     1,
			body:       `{"current_password":"` + studentPassword + `","new_password":"a-brand-new-password","confirm_password":"a-brand-new-password"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong current password",
			userID:     1,
			body:       `{"current_password":"not-the-password","new_password":"a-brand-new-password","confirm_password":"a-brand-new-password"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageWrongPassword,
		},
		{
			name:       "the new password repeats the old one",
			userID:     1,
			body:       `{"current_password":"` + studentPassword + `","new_password":"` + studentPassword + `","confirm_password":"` + studentPassword + `"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageSamePassword,
		},
		{
			name:       "confirmation does not match",
			userID:     1,
			body:       `{"current_password":"` + studentPassword + `","new_password":"a-brand-new-password","confirm_password":"another-password"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "New password and the confirmation do not match.",
		},
		{
			name:       "short new password",
			userID:     1,
			body:       `{"current_password":"` + studentPassword + `","new_password":"short","confirm_password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "New password must be at least 8 characters long.",
		},
		{
			name:       "missing current password",
			userID:     1,
			body:       `{"current_password":"","new_password":"a-brand-new-password","confirm_password":"a-brand-new-password"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Current password is required.",
		},
		{
			name:       "unknown account",
			userID:     99,
			body:       `{"current_password":"` + studentPassword + `","new_password":"a-brand-new-password","confirm_password":"a-brand-new-password"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid body",
			userID:     1,
			body:       `{"current_password":}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles := newProfileStore(t)
			handler := newProfileHandler(t, profiles, testPhotos(t))
			request := jsonRequest(t, http.MethodPost, "/api/profile/password", test.body, test.userID, auth.RoleStudent)
			rec := serve(handler.ChangePassword(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				if test.wantError != "" {
					if got := errorMessage(t, rec); got != test.wantError {
						t.Errorf("error = %q, want %q", got, test.wantError)
					}
				}
				if len(profiles.passwords) != 0 {
					t.Error("the store received a password although the request was refused")
				}
				return
			}
			if len(profiles.passwords) != 1 {
				t.Fatalf("the store received %d passwords, want 1", len(profiles.passwords))
			}
			if auth.CheckPassword(profiles.passwords[0], "a-brand-new-password") != nil {
				t.Error("the store did not receive the hash of the new password")
			}
			if strings.Contains(rec.Body.String(), "a-brand-new-password") {
				t.Error("the response leaks the new password")
			}
		})
	}
}

func TestPeopleCard(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		pathValue  string
		role       string
		wantStatus int
		wantFields map[string]any
	}{
		{
			name:       "a student seen by a teacher",
			pathValue:  "1",
			role:       auth.RoleTeacher,
			wantStatus: http.StatusOK,
			wantFields: map[string]any{
				"user_id":     float64(1),
				"role":        "student",
				"name":        "Active Student",
				"batch":       "CSE-23",
				"section":     "A",
				"email":       "2023331001@student.sust.edu",
				"phone":       "01712345678",
				"blood_group": "O+",
				"photo_url":   "/uploads/photos/aaaa.png",
			},
		},
		{
			name:       "a class representative seen by a student",
			pathValue:  "2",
			role:       auth.RoleStudent,
			wantStatus: http.StatusOK,
			wantFields: map[string]any{"role": "cr", "section": "B"},
		},
		{
			name:       "a teacher seen by a student",
			pathValue:  "6",
			role:       auth.RoleStudent,
			wantStatus: http.StatusOK,
			wantFields: map[string]any{
				"role":        "teacher",
				"designation": "Assistant Professor",
				"office_room": "AB-401",
			},
		},
		{name: "an admin is not a person card", pathValue: "7", role: auth.RoleStudent, wantStatus: http.StatusNotFound},
		{name: "an unknown account", pathValue: "99", role: auth.RoleAdmin, wantStatus: http.StatusNotFound},
		{name: "not a number", pathValue: "abc", role: auth.RoleStudent, wantStatus: http.StatusBadRequest},
		{name: "zero", pathValue: "0", role: auth.RoleStudent, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles := newProfileStore(t)
			handler := newProfileHandler(t, profiles, testPhotos(t))
			request := jsonRequest(t, http.MethodGet, "/api/people/"+test.pathValue, "", 11, test.role)
			request.SetPathValue("user_id", test.pathValue)
			rec := serve(handler.Person(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			payload := decodeBody(t, rec)
			for name, want := range test.wantFields {
				if got := payload[name]; got != want {
					t.Errorf("%s = %v, want %v", name, got, want)
				}
			}
			for _, hidden := range []string{"date_of_birth", "status", "reg_no", "password_hash"} {
				if _, present := payload[hidden]; present {
					t.Errorf("the person card carries the %q field", hidden)
				}
			}
		})
	}
}

func TestProfilePatchRejectsALargePhoto(t *testing.T) {
	silenceLog(t)
	profiles := newProfileStore(t)
	photos := testPhotos(t)
	handler := newProfileHandler(t, profiles, photos)
	oversized := make([]byte, upload.MaxPhotoBytes+1<<16+1)
	copy(oversized, pngPhoto(t, 240, 240))
	request := multipartRequest(t, http.MethodPatch, "/api/profile", map[string][]string{},
		formFile{field: "photo", name: "me.png", data: oversized})
	request = withCurrentUser(request, 1, auth.RoleStudent)
	rec := serve(handler.Patch(), request)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != MessagePhotoTooLarge {
		t.Errorf("error = %q, want %q", got, MessagePhotoTooLarge)
	}
}
