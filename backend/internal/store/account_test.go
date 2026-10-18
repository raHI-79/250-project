package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func seedRoutineCourse(t *testing.T, ctx context.Context, store *Store, pool *pgxpool.Pool, code, title string) int64 {
	t.Helper()
	course, err := store.CreateCourse(ctx, code, title)
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	teacherUserID := insertUser(t, ctx, pool, "teacher", "T-"+code, StatusActive, 0)
	var teacherID int64
	query := `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, $2, 'Routine Teacher', $2 || '@sust.edu', '01800000000') RETURNING id`
	if err := pool.QueryRow(ctx, query, teacherUserID, "T-"+code).Scan(&teacherID); err != nil {
		t.Fatalf("could not insert the routine teacher: %v", err)
	}
	routine := `INSERT INTO routines (batch, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type)
VALUES ('CSE-22', 'A', $1, $2, 1, '08:00', '09:00', '1003', 'class')`
	if _, err := pool.Exec(ctx, routine, course.ID, teacherID); err != nil {
		t.Fatalf("could not insert the routine entry: %v", err)
	}
	return course.ID
}

func TestSearchDropCourseOptions(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	routineCourse := seedRoutineCourse(t, ctx, store, pool, "CSE-245", "Databases")
	if _, err := store.CreateCourse(ctx, "CSE-299", "Project Work"); err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name      string
		search    string
		wantCount int
		wantID    int64
	}{
		{name: "no search", search: "", wantCount: 1, wantID: routineCourse},
		{name: "by code", search: "CSE-245", wantCount: 1, wantID: routineCourse},
		{name: "by code without a dash", search: "245", wantCount: 1, wantID: routineCourse},
		{name: "by title", search: "data", wantCount: 1, wantID: routineCourse},
		{name: "by lowercase title", search: "databases", wantCount: 1, wantID: routineCourse},
		{name: "a course without a routine", search: "Project", wantCount: 0},
		{name: "no match", search: "zzz", wantCount: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			courses, err := store.SearchDropCourseOptions(ctx, tc.search)
			if err != nil {
				t.Fatalf("SearchDropCourseOptions() returned an unexpected error: %v", err)
			}
			if len(courses) != tc.wantCount {
				t.Fatalf("got %d courses, want %d (%+v)", len(courses), tc.wantCount, courses)
			}
			if tc.wantCount > 0 && courses[0].ID != tc.wantID {
				t.Errorf("course id = %d, want %d", courses[0].ID, tc.wantID)
			}
		})
	}
}

func TestFilterDropCourseIDs(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	routineCourse := seedRoutineCourse(t, ctx, store, pool, "CSE-245", "Databases")
	plain, err := store.CreateCourse(ctx, "CSE-299", "Project Work")
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name      string
		ids       []int64
		wantCount int
	}{
		{name: "no ids", ids: nil, wantCount: 0},
		{name: "a routine course", ids: []int64{routineCourse}, wantCount: 1},
		{name: "a course without a routine", ids: []int64{plain.ID}, wantCount: 0},
		{name: "a mix", ids: []int64{routineCourse, plain.ID}, wantCount: 1},
		{name: "an unknown id", ids: []int64{999999}, wantCount: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			allowed, err := store.FilterDropCourseIDs(ctx, tc.ids)
			if err != nil {
				t.Fatalf("FilterDropCourseIDs() returned an unexpected error: %v", err)
			}
			if len(allowed) != tc.wantCount {
				t.Errorf("allowed = %v, want %d of them", allowed, tc.wantCount)
			}
		})
	}
}

func TestEmailVerification(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	registration := newRegistration("2023331300", "verify@student.sust.edu")
	registration.Status = StatusUnverified
	registration.CodeHash = "stored-code-hash"
	expiresAt := time.Now().Add(15 * time.Minute)
	sentAt := time.Now()
	registration.CodeExpiresAt = &expiresAt
	registration.CodeSentAt = &sentAt
	result, err := store.CreateRegistration(ctx, registration)
	if err != nil {
		t.Fatalf("CreateRegistration() returned an unexpected error: %v", err)
	}
	challenge, err := store.FindEmailChallenge(ctx, "2023331300")
	if err != nil {
		t.Fatalf("FindEmailChallenge() returned an unexpected error: %v", err)
	}
	if challenge.UserID != result.UserID || challenge.Status != StatusUnverified {
		t.Errorf("challenge = %+v, want the unverified account", challenge)
	}
	if challenge.CodeHash != "stored-code-hash" || challenge.Email != "verify@student.sust.edu" {
		t.Errorf("challenge = %+v, want the stored code and email", challenge)
	}
	if challenge.Attempts != 0 || challenge.ExpiresAt == nil || challenge.SentAt == nil {
		t.Errorf("challenge = %+v, want no attempts and both timestamps", challenge)
	}
	attempts, err := store.CountEmailAttempt(ctx, result.UserID)
	if err != nil || attempts != 1 {
		t.Fatalf("CountEmailAttempt() = %d, %v, want 1 and no error", attempts, err)
	}
	if err := store.ClearEmailCode(ctx, result.UserID); err != nil {
		t.Fatalf("ClearEmailCode() returned an unexpected error: %v", err)
	}
	cleared, err := store.FindEmailChallenge(ctx, "2023331300")
	if err != nil {
		t.Fatalf("FindEmailChallenge() returned an unexpected error: %v", err)
	}
	if cleared.CodeHash != "" || cleared.ExpiresAt != nil || cleared.Attempts != 0 {
		t.Errorf("challenge after clearing = %+v, want no code", cleared)
	}
	if err := store.StartEmailVerification(ctx, result.UserID, "second-hash", expiresAt, sentAt); err != nil {
		t.Fatalf("StartEmailVerification() returned an unexpected error: %v", err)
	}
	restarted, err := store.FindEmailChallenge(ctx, "2023331300")
	if err != nil {
		t.Fatalf("FindEmailChallenge() returned an unexpected error: %v", err)
	}
	if restarted.CodeHash != "second-hash" {
		t.Errorf("code hash = %q, want second-hash", restarted.CodeHash)
	}
	if err := store.ConfirmEmail(ctx, result.UserID, time.Now()); err != nil {
		t.Fatalf("ConfirmEmail() returned an unexpected error: %v", err)
	}
	user, err := store.GetAuthUser(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	if user.Status != StatusPending {
		t.Errorf("status = %q, want %q", user.Status, StatusPending)
	}
	if err := store.ConfirmEmail(ctx, result.UserID, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Errorf("second ConfirmEmail() error = %v, want ErrNotFound", err)
	}
	if _, err := store.FindEmailChallenge(ctx, "2023339999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindEmailChallenge() for an unknown student error = %v, want ErrNotFound", err)
	}
}

func TestPromoteCR(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	activeID := insertUser(t, ctx, pool, "student", "2023331400", StatusActive, 0)
	insertStudent(t, ctx, pool, activeID, "2023331400", false)
	crID := insertUser(t, ctx, pool, "student", "2023331401", StatusActive, 0)
	insertStudent(t, ctx, pool, crID, "2023331401", true)
	pendingID := insertUser(t, ctx, pool, "student", "2023331402", StatusPending, 0)
	insertStudent(t, ctx, pool, pendingID, "2023331402", false)
	teacherID := insertUser(t, ctx, pool, "teacher", "T-400", StatusActive, 0)
	insertTeacher(t, ctx, pool, teacherID, "T-400")
	tests := []struct {
		name    string
		userID  int64
		wantErr error
	}{
		{name: "active student", userID: activeID},
		{name: "already a class representative", userID: crID, wantErr: ErrAlreadyCR},
		{name: "pending student", userID: pendingID, wantErr: ErrNotFound},
		{name: "teacher", userID: teacherID, wantErr: ErrNotFound},
		{name: "unknown account", userID: 999999, wantErr: ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			promoted, err := store.PromoteCR(ctx, tc.userID)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("PromoteCR() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("PromoteCR() returned an unexpected error: %v", err)
			}
			if promoted.Batch != "CSE-23" || promoted.Section != "A" || promoted.FullName == "" {
				t.Errorf("promoted = %+v, want the batch, section and name", promoted)
			}
			user, err := store.GetAuthUser(ctx, tc.userID)
			if err != nil {
				t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
			}
			if !user.IsCR {
				t.Error("the student is still not a class representative")
			}
			if user.TokenVersion != 1 {
				t.Errorf("token version = %d, want 1", user.TokenVersion)
			}
		})
	}
}

func TestTeacherInvitations(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	expiresAt := time.Now().Add(72 * time.Hour).Truncate(time.Second)
	teacher := NewTeacher{
		Name:            "Invited Teacher",
		TeacherID:       "T-500",
		Designation:     "Assistant Professor",
		Email:           "invited@sust.edu",
		Phone:           "01812345678",
		BloodGroup:      "B+",
		DateOfBirth:     "1988-03-09",
		OfficeRoom:      "AB-401",
		JoiningDate:     "2015-01-15",
		PhotoPath:       "/uploads/photos/teacher.png",
		InviteHash:      "stored-invite-hash",
		InviteExpiresAt: expiresAt,
	}
	userID, err := store.CreateTeacher(ctx, teacher)
	if err != nil {
		t.Fatalf("CreateTeacher() returned an unexpected error: %v", err)
	}
	user, err := store.GetAuthUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	if user.Status != StatusInvited {
		t.Errorf("status = %q, want %q", user.Status, StatusInvited)
	}
	if user.PhotoPath != "/uploads/photos/teacher.png" {
		t.Errorf("photo = %q, want the stored photo", user.PhotoPath)
	}
	if hash := storedPasswordHash(t, ctx, pool, "T-500"); hash != "" {
		t.Errorf("the invited teacher has the password hash %q, want none", hash)
	}
	invite, err := store.FindInvite(ctx, "stored-invite-hash")
	if err != nil {
		t.Fatalf("FindInvite() returned an unexpected error: %v", err)
	}
	if invite.UserID != userID || invite.TeacherID != "T-500" || invite.DateOfBirth != "1988-03-09" {
		t.Errorf("invite = %+v, want the stored teacher", invite)
	}
	if _, err := store.FindInvite(ctx, "another-hash"); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindInvite() with a wrong hash error = %v, want ErrNotFound", err)
	}
	attempts, err := store.CountInviteAttempt(ctx, userID)
	if err != nil || attempts != 1 {
		t.Fatalf("CountInviteAttempt() = %d, %v, want 1 and no error", attempts, err)
	}
	refreshed, err := store.ResetTeacherInvite(ctx, userID, "second-invite-hash", expiresAt)
	if err != nil {
		t.Fatalf("ResetTeacherInvite() returned an unexpected error: %v", err)
	}
	if refreshed.Email != "invited@sust.edu" || refreshed.Attempts != 0 {
		t.Errorf("refreshed invite = %+v, want the email and no attempts", refreshed)
	}
	if _, err := store.FindInvite(ctx, "stored-invite-hash"); !errors.Is(err, ErrNotFound) {
		t.Errorf("the old invitation hash still works: %v", err)
	}
	update := TeacherUpdate{}
	designation := "Professor"
	update.Designation = &designation
	if err := store.UpdateTeacher(ctx, userID, update); err != nil {
		t.Fatalf("UpdateTeacher() returned an unexpected error: %v", err)
	}
	teachers, err := store.ListTeachers(ctx, TeacherFilter{Status: StatusFilterAll})
	if err != nil {
		t.Fatalf("ListTeachers() returned an unexpected error: %v", err)
	}
	if len(teachers) != 1 || teachers[0].Designation != "Professor" {
		t.Fatalf("teachers = %+v, want the updated designation", teachers)
	}
	if teachers[0].OfficeRoom != "AB-401" || teachers[0].JoiningDate != "2015-01-15" {
		t.Errorf("teacher row = %+v, want the office room and joining date", teachers[0])
	}
	if err := store.ActivateTeacher(ctx, userID, "new-password-hash", ""); err != nil {
		t.Fatalf("ActivateTeacher() returned an unexpected error: %v", err)
	}
	active, err := store.GetAuthUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	if active.Status != StatusActive {
		t.Errorf("status = %q, want %q", active.Status, StatusActive)
	}
	if _, err := store.FindInvite(ctx, "second-invite-hash"); !errors.Is(err, ErrNotFound) {
		t.Errorf("the invitation hash survived the activation: %v", err)
	}
	if err := store.ActivateTeacher(ctx, userID, "another-hash", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("second ActivateTeacher() error = %v, want ErrNotFound", err)
	}
	if _, err := store.ResetTeacherInvite(ctx, userID, "third-hash", expiresAt); !errors.Is(err, ErrNotInvited) {
		t.Errorf("ResetTeacherInvite() for an active teacher error = %v, want ErrNotInvited", err)
	}
	if _, err := store.ResetTeacherInvite(ctx, 999999, "third-hash", expiresAt); !errors.Is(err, ErrNotFound) {
		t.Errorf("ResetTeacherInvite() for an unknown teacher error = %v, want ErrNotFound", err)
	}
}

func TestProfilesAndPeople(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	courseID := seedRoutineCourse(t, ctx, store, pool, "CSE-245", "Databases")
	registration := newRegistration("2023331600", "profile@student.sust.edu")
	registration.DropCourseIDs = []int64{courseID}
	result, err := store.CreateRegistration(ctx, registration)
	if err != nil {
		t.Fatalf("CreateRegistration() returned an unexpected error: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status = 'active' WHERE id = $1", result.UserID); err != nil {
		t.Fatalf("could not activate the student: %v", err)
	}
	profile, err := store.GetProfile(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetProfile() returned an unexpected error: %v", err)
	}
	if profile.Role != "student" || profile.RegNo != "2023331600" || profile.Batch != "CSE-23" {
		t.Errorf("profile = %+v, want the student details", profile)
	}
	if profile.DateOfBirth != "2003-05-14" || profile.PhotoPath == "" {
		t.Errorf("profile = %+v, want the birth date and photo", profile)
	}
	if len(profile.DropCourses) != 1 || profile.DropCourses[0].Code != "CSE-245" {
		t.Errorf("drop courses = %+v, want CSE-245", profile.DropCourses)
	}
	phone := "01799999999"
	newPhoto := "/uploads/photos/replacement.png"
	emptySet := []int64{}
	oldPhoto, err := store.UpdateProfile(ctx, result.UserID, "student", ProfileUpdate{
		Phone:         &phone,
		PhotoPath:     &newPhoto,
		DropCourseIDs: &emptySet,
	})
	if err != nil {
		t.Fatalf("UpdateProfile() returned an unexpected error: %v", err)
	}
	if oldPhoto != registration.PhotoPath {
		t.Errorf("old photo = %q, want %q", oldPhoto, registration.PhotoPath)
	}
	updated, err := store.GetProfile(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetProfile() returned an unexpected error: %v", err)
	}
	if updated.Phone != phone || updated.PhotoPath != newPhoto {
		t.Errorf("profile = %+v, want the new phone and photo", updated)
	}
	if len(updated.DropCourses) != 0 {
		t.Errorf("drop courses = %+v, want an empty set", updated.DropCourses)
	}
	back := []int64{courseID}
	if _, err := store.UpdateProfile(ctx, result.UserID, "student", ProfileUpdate{DropCourseIDs: &back}); err != nil {
		t.Fatalf("UpdateProfile() returned an unexpected error: %v", err)
	}
	restored, err := store.GetProfile(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetProfile() returned an unexpected error: %v", err)
	}
	if len(restored.DropCourses) != 1 {
		t.Errorf("drop courses = %+v, want the course back", restored.DropCourses)
	}
	if _, err := store.SetPassword(ctx, result.UserID, "another-hash"); err != nil {
		t.Fatalf("SetPassword() returned an unexpected error: %v", err)
	}
	hash, err := store.GetPasswordHash(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetPasswordHash() returned an unexpected error: %v", err)
	}
	if hash != "another-hash" {
		t.Errorf("password hash = %q, want another-hash", hash)
	}
	afterPassword, err := store.GetAuthUser(ctx, result.UserID)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	if afterPassword.TokenVersion != 1 {
		t.Errorf("token version = %d, want 1", afterPassword.TokenVersion)
	}
	card, err := store.FindPerson(ctx, result.UserID)
	if err != nil {
		t.Fatalf("FindPerson() returned an unexpected error: %v", err)
	}
	if card.Role != "student" || card.Name == "" || card.Batch != "CSE-23" {
		t.Errorf("card = %+v, want the student card", card)
	}
	if _, err := store.PromoteCR(ctx, result.UserID); err != nil {
		t.Fatalf("PromoteCR() returned an unexpected error: %v", err)
	}
	asCR, err := store.FindPerson(ctx, result.UserID)
	if err != nil {
		t.Fatalf("FindPerson() returned an unexpected error: %v", err)
	}
	if asCR.Role != "cr" {
		t.Errorf("card role = %q, want cr", asCR.Role)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status = 'removed' WHERE id = $1", result.UserID); err != nil {
		t.Fatalf("could not remove the student: %v", err)
	}
	if _, err := store.FindPerson(ctx, result.UserID); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindPerson() for a removed account error = %v, want ErrNotFound", err)
	}
	adminID := insertUser(t, ctx, pool, "admin", "admin", StatusActive, 0)
	if _, err := store.FindPerson(ctx, adminID); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindPerson() for an admin error = %v, want ErrNotFound", err)
	}
	adminEmail := "office@sust.edu"
	adminPhone := "01888888888"
	if _, err := store.UpdateProfile(ctx, adminID, "admin", ProfileUpdate{
		ContactEmail: &adminEmail,
		ContactPhone: &adminPhone,
	}); err != nil {
		t.Fatalf("UpdateProfile() for the admin returned an unexpected error: %v", err)
	}
	adminProfile, err := store.GetProfile(ctx, adminID)
	if err != nil {
		t.Fatalf("GetProfile() returned an unexpected error: %v", err)
	}
	if adminProfile.ContactEmail != adminEmail || adminProfile.ContactPhone != adminPhone {
		t.Errorf("admin profile = %+v, want the new contact details", adminProfile)
	}
	if _, err := store.GetProfile(ctx, 999999); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetProfile() for an unknown account error = %v, want ErrNotFound", err)
	}
	if _, err := store.UpdateProfile(ctx, adminID, "principal", ProfileUpdate{}); !errors.Is(err, ErrUnknownRole) {
		t.Errorf("UpdateProfile() with an unknown role error = %v, want ErrUnknownRole", err)
	}
}
