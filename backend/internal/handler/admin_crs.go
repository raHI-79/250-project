package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

type createCRRequest struct {
	FullName        string `json:"full_name"`
	RegNo           string `json:"reg_no"`
	Batch           string `json:"batch"`
	Section         string `json:"section"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	BloodGroup      string `json:"blood_group"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type createdResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

type crsResponse struct {
	Items []studentItem `json:"items"`
}

func (a *Admin) CreateCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request createCRRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		cr, err := request.toNewCR()
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		hash, err := auth.HashPassword(request.Password)
		if err != nil {
			log.Printf("handler: could not hash a class representative password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		cr.PasswordHash = hash
		userID, err := a.people.CreateCR(r.Context(), cr)
		if err != nil {
			if errors.Is(err, store.ErrDuplicateAccount) {
				httpx.WriteError(w, http.StatusConflict, MessageDuplicateAccount)
				return
			}
			log.Printf("handler: could not create a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, createdResponse{ID: userID, Message: MessageCRCreated})
	}
}

func (request createCRRequest) toNewCR() (store.NewCR, error) {
	var cr store.NewCR
	fields := []struct {
		target *string
		value  string
		label  string
		check  func(string, string) (string, error)
	}{
		{target: &cr.FullName, value: request.FullName, label: "Full name", check: validate.Required},
		{target: &cr.RegNo, value: request.RegNo, label: "Registration number", check: validate.Required},
		{target: &cr.Batch, value: request.Batch, label: "Batch", check: validate.Batch},
		{target: &cr.Section, value: request.Section, label: "Section", check: validate.Section},
		{target: &cr.Email, value: request.Email, label: "Email", check: validate.Email},
		{target: &cr.Phone, value: request.Phone, label: "Phone", check: validate.Phone},
		{target: &cr.BloodGroup, value: request.BloodGroup, label: "Blood group", check: validate.BloodGroup},
	}
	for _, field := range fields {
		value, err := field.check(field.label, field.value)
		if err != nil {
			return store.NewCR{}, err
		}
		*field.target = value
	}
	if err := validate.Password("Password", request.Password, request.ConfirmPassword); err != nil {
		return store.NewCR{}, err
	}
	return cr, nil
}

func (a *Admin) CRs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		filter := store.CRFilter{Search: strings.TrimSpace(query.Get("search"))}
		if batch := strings.TrimSpace(query.Get("batch")); batch != "" {
			value, err := validate.Batch("batch", batch)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			filter.Batch = value
		}
		crs, err := a.people.ListCRs(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the class representatives: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, crsResponse{Items: studentItems(crs)})
	}
}

func (a *Admin) DemoteCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.DemoteCR(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageStudentNotFound)
		case errors.Is(err, store.ErrNotCR):
			httpx.WriteError(w, http.StatusConflict, MessageNotCR)
		case err != nil:
			log.Printf("handler: could not demote a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageCRDemoted})
		}
	}
}

func (a *Admin) DeactivateCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.DeactivateCR(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageCRNotFound)
		case errors.Is(err, store.ErrAlreadyRemoved):
			httpx.WriteError(w, http.StatusConflict, MessageAlreadyRemoved)
		case err != nil:
			log.Printf("handler: could not deactivate a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageCRDeactivated})
		}
	}
}
