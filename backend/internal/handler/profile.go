package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	MessageProfileMissing  = "No profile found for this account."
	MessageProfileUpdated  = "Profile updated."
	MessageProfileField    = "This field cannot be changed from the profile page."
	MessageNothingToUpdate = "Send at least one field to change."
	MessagePersonMissing   = "No person found with this ID."
	MessageWrongPassword   = "The current password is wrong."
	MessageSamePassword    = "The new password must be different from the current one."
	MaxBioLength           = 500
	fieldOfficeRoom        = "office_room"
	fieldBio               = "bio"
	fieldContactEmail      = "contact_email"
	fieldContactPhone      = "contact_phone"
)

type ProfileStore interface {
	GetProfile(ctx context.Context, userID int64) (store.Profile, error)
	UpdateProfile(ctx context.Context, userID int64, role string, update store.ProfileUpdate) (string, error)
	GetPasswordHash(ctx context.Context, userID int64) (string, error)
	SetPassword(ctx context.Context, userID int64, passwordHash string) (int, error)
	FilterDropCourseIDs(ctx context.Context, courseIDs []int64) ([]int64, error)
	FindPerson(ctx context.Context, userID int64) (store.PersonCard, error)
}

type Profile struct {
	cfg      *config.Config
	profiles ProfileStore
	photos   PhotoStore
}

func NewProfile(cfg *config.Config, profiles ProfileStore, photos PhotoStore) *Profile {
	return &Profile{cfg: cfg, profiles: profiles, photos: photos}
}

type profileCourse struct {
	ID    int64  `json:"id"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type profileResponse struct {
	UserID       int64           `json:"user_id"`
	Role         string          `json:"role"`
	Status       string          `json:"status"`
	CreatedAt    string          `json:"created_at"`
	PhotoURL     string          `json:"photo_url,omitempty"`
	FullName     string          `json:"full_name,omitempty"`
	RegNo        string          `json:"reg_no,omitempty"`
	Batch        string          `json:"batch,omitempty"`
	Section      string          `json:"section,omitempty"`
	Email        string          `json:"email,omitempty"`
	Phone        string          `json:"phone,omitempty"`
	BloodGroup   string          `json:"blood_group,omitempty"`
	DateOfBirth  string          `json:"date_of_birth,omitempty"`
	IsCR         *bool           `json:"is_cr,omitempty"`
	DropCourses  []profileCourse `json:"drop_courses,omitempty"`
	Name         string          `json:"name,omitempty"`
	Designation  string          `json:"designation,omitempty"`
	OfficeRoom   string          `json:"office_room,omitempty"`
	JoiningDate  string          `json:"joining_date,omitempty"`
	Bio          string          `json:"bio,omitempty"`
	Identifier   string          `json:"identifier,omitempty"`
	ContactEmail string          `json:"contact_email,omitempty"`
	ContactPhone string          `json:"contact_phone,omitempty"`
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type personResponse struct {
	UserID      int64  `json:"user_id"`
	Role        string `json:"role"`
	Name        string `json:"name"`
	Batch       string `json:"batch,omitempty"`
	Section     string `json:"section,omitempty"`
	Designation string `json:"designation,omitempty"`
	OfficeRoom  string `json:"office_room,omitempty"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	BloodGroup  string `json:"blood_group,omitempty"`
	PhotoURL    string `json:"photo_url,omitempty"`
}

func (p *Profile) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		profile, err := p.profiles.GetProfile(r.Context(), current.ID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageProfileMissing)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a profile: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, profileBody(current.Role, profile))
	}
}

func profileBody(role string, profile store.Profile) profileResponse {
	body := profileResponse{
		UserID:    profile.UserID,
		Role:      role,
		Status:    profile.Status,
		CreatedAt: timestamp(profile.CreatedAt),
		PhotoURL:  profile.PhotoPath,
	}
	switch role {
	case auth.RoleStudent, auth.RoleCR:
		isCR := profile.IsCR
		body.FullName = profile.FullName
		body.RegNo = profile.RegNo
		body.Batch = profile.Batch
		body.Section = profile.Section
		body.Email = profile.Email
		body.Phone = profile.Phone
		body.BloodGroup = profile.BloodGroup
		body.DateOfBirth = profile.DateOfBirth
		body.IsCR = &isCR
		body.DropCourses = profileCourses(profile.DropCourses)
	case auth.RoleTeacher:
		body.Name = profile.FullName
		body.Identifier = profile.TeacherID
		body.Designation = profile.Designation
		body.Email = profile.Email
		body.Phone = profile.Phone
		body.BloodGroup = profile.BloodGroup
		body.DateOfBirth = profile.DateOfBirth
		body.OfficeRoom = profile.OfficeRoom
		body.JoiningDate = profile.JoiningDate
		body.Bio = profile.Bio
	case auth.RoleAdmin:
		body.Identifier = profile.Identifier
		body.ContactEmail = profile.ContactEmail
		body.ContactPhone = profile.ContactPhone
	}
	return body
}

func profileCourses(courses []store.Course) []profileCourse {
	items := make([]profileCourse, 0, len(courses))
	for _, course := range courses {
		items = append(items, profileCourse{ID: course.ID, Code: course.Code, Title: course.Title})
	}
	return items
}

func (p *Profile) Patch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		if err := p.photos.Parse(r); err != nil {
			status, message := photoError(err)
			httpx.WriteError(w, status, message)
			return
		}
		update, err := p.updateFrom(r, current.Role)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		photoPath, err := p.photos.Save(r)
		if err != nil {
			status, message := photoError(err)
			httpx.WriteError(w, status, message)
			return
		}
		if photoPath != "" {
			update.PhotoPath = &photoPath
		}
		if emptyUpdate(update) {
			httpx.WriteError(w, http.StatusBadRequest, MessageNothingToUpdate)
			return
		}
		oldPhoto, err := p.profiles.UpdateProfile(r.Context(), current.ID, current.Role, update)
		if err != nil {
			p.removePhoto(photoPath)
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusNotFound, MessageProfileMissing)
				return
			}
			log.Printf("handler: could not update a profile: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if oldPhoto != "" && oldPhoto != photoPath {
			p.removePhoto(oldPhoto)
		}
		profile, err := p.profiles.GetProfile(r.Context(), current.ID)
		if err != nil {
			log.Printf("handler: could not reload a profile: %v", err)
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageProfileUpdated})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, profileBody(current.Role, profile))
	}
}

func (p *Profile) updateFrom(r *http.Request, role string) (store.ProfileUpdate, error) {
	var update store.ProfileUpdate
	allowed := allowedProfileFields(role)
	for name := range r.MultipartForm.Value {
		if !allowed[name] {
			return store.ProfileUpdate{}, errors.New(MessageProfileField)
		}
	}
	if raw, ok := r.MultipartForm.Value[fieldPhone]; ok && len(raw) > 0 {
		phone, err := validate.Phone("Phone", raw[0])
		if err != nil {
			return store.ProfileUpdate{}, err
		}
		update.Phone = &phone
	}
	switch role {
	case auth.RoleStudent, auth.RoleCR:
		if raw, ok := r.MultipartForm.Value[fieldDropCourseIDs]; ok {
			courseIDs, err := p.dropCourseIDs(r, raw)
			if err != nil {
				return store.ProfileUpdate{}, err
			}
			update.DropCourseIDs = &courseIDs
		}
	case auth.RoleTeacher:
		if raw, ok := r.MultipartForm.Value[fieldOfficeRoom]; ok && len(raw) > 0 {
			room, err := validate.Required("Office room", raw[0])
			if err != nil {
				return store.ProfileUpdate{}, err
			}
			update.OfficeRoom = &room
		}
		if raw, ok := r.MultipartForm.Value[fieldBio]; ok && len(raw) > 0 {
			bio := strings.TrimSpace(raw[0])
			if len([]rune(bio)) > MaxBioLength {
				return store.ProfileUpdate{}, errors.New("Bio must be at most 500 characters long.")
			}
			update.Bio = &bio
		}
	case auth.RoleAdmin:
		if raw, ok := r.MultipartForm.Value[fieldContactEmail]; ok && len(raw) > 0 {
			email, err := validate.Email("Contact email", raw[0])
			if err != nil {
				return store.ProfileUpdate{}, err
			}
			update.ContactEmail = &email
		}
		if raw, ok := r.MultipartForm.Value[fieldContactPhone]; ok && len(raw) > 0 {
			phone, err := validate.Phone("Contact phone", raw[0])
			if err != nil {
				return store.ProfileUpdate{}, err
			}
			update.ContactPhone = &phone
		}
	default:
		return store.ProfileUpdate{}, errors.New(MessageProfileField)
	}
	return update, nil
}

func allowedProfileFields(role string) map[string]bool {
	switch role {
	case auth.RoleStudent, auth.RoleCR:
		return map[string]bool{fieldPhone: true, fieldDropCourseIDs: true}
	case auth.RoleTeacher:
		return map[string]bool{fieldPhone: true, fieldOfficeRoom: true, fieldBio: true}
	case auth.RoleAdmin:
		return map[string]bool{fieldContactEmail: true, fieldContactPhone: true}
	default:
		return map[string]bool{}
	}
}

func emptyUpdate(update store.ProfileUpdate) bool {
	return update.Phone == nil && update.PhotoPath == nil && update.OfficeRoom == nil &&
		update.Bio == nil && update.ContactEmail == nil && update.ContactPhone == nil &&
		update.DropCourseIDs == nil
}

func (p *Profile) dropCourseIDs(r *http.Request, raw []string) ([]int64, error) {
	wanted := make([]int64, 0, len(raw))
	seen := make(map[int64]bool, len(raw))
	for _, value := range raw {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			courseID, err := strconv.ParseInt(part, 10, 64)
			if err != nil || courseID <= 0 {
				return nil, errors.New(MessageUnknownDropCourse)
			}
			if seen[courseID] {
				continue
			}
			seen[courseID] = true
			wanted = append(wanted, courseID)
		}
	}
	if len(wanted) == 0 {
		return []int64{}, nil
	}
	allowed, err := p.profiles.FilterDropCourseIDs(r.Context(), wanted)
	if err != nil {
		log.Printf("handler: could not check the drop course choices: %v", err)
		return nil, errors.New(MessageUnknownDropCourse)
	}
	if len(allowed) != len(wanted) {
		return nil, errors.New(MessageUnknownDropCourse)
	}
	return wanted, nil
}

func (p *Profile) removePhoto(photoPath string) {
	if photoPath == "" {
		return
	}
	if err := p.photos.Delete(photoPath); err != nil {
		log.Printf("handler: could not delete a profile photo: %v", err)
	}
}

func (p *Profile) ChangePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request passwordRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if request.CurrentPassword == "" {
			httpx.WriteError(w, http.StatusBadRequest, "Current password is required.")
			return
		}
		if err := validate.Password("New password", request.NewPassword, request.ConfirmPassword); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if request.NewPassword == request.CurrentPassword {
			httpx.WriteError(w, http.StatusBadRequest, MessageSamePassword)
			return
		}
		hash, err := p.profiles.GetPasswordHash(r.Context(), current.ID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageSessionExpired)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if auth.CheckPassword(hash, request.CurrentPassword) != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessageWrongPassword)
			return
		}
		newHash, err := auth.HashPassword(request.NewPassword)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessagePasswordTooLong)
			return
		}
		if _, err := p.profiles.SetPassword(r.Context(), current.ID, newHash); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageSessionExpired)
				return
			}
			log.Printf("handler: could not change a password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePasswordUpdated})
	}
}

func (p *Profile) Person() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("user_id")), 10, 64)
		if err != nil || userID <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
			return
		}
		card, err := p.profiles.FindPerson(r.Context(), userID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessagePersonMissing)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a person: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, personResponse{
			UserID:      card.UserID,
			Role:        card.Role,
			Name:        card.Name,
			Batch:       card.Batch,
			Section:     card.Section,
			Designation: card.Designation,
			OfficeRoom:  card.OfficeRoom,
			Email:       card.Email,
			Phone:       card.Phone,
			BloodGroup:  card.BloodGroup,
			PhotoURL:    card.PhotoPath,
		})
	}
}
