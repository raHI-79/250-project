package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"time"
)

const (
	MessageDriveLinkSaved   = "Drive link saved."
	MessageDriveLinkMissing = "Your batch has no Drive link yet."
)

type DriveLinkStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	GetBatchDriveLink(ctx context.Context, batch string) (store.DriveLinkRow, error)
	SaveBatchDriveLink(ctx context.Context, link store.NewDriveLink) (store.DriveLinkRow, error)
	DeleteBatchDriveLink(ctx context.Context, batch string) error
	ListBatchDriveLinks(ctx context.Context) ([]store.DriveLinkRow, error)
}

type DriveLink struct {
	links DriveLinkStore
}

func NewDriveLink(links DriveLinkStore) *DriveLink {
	return &DriveLink{links: links}
}

type driveLinkRequest struct {
	Link string `json:"link"`
}

type driveLinkItem struct {
	Batch     string `json:"batch"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	UpdatedAt string `json:"updated_at,omitempty"`
	IsMine    bool   `json:"is_mine"`
}

type driveLinksResponse struct {
	Items []driveLinkItem `json:"items"`
}

type driveLinkResponse struct {
	driveLinkItem
	Message string `json:"message,omitempty"`
}

func (d *DriveLink) CRGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := crProfile(w, r, d.links.GetStudentProfile)
		if !ok {
			return
		}
		link, err := d.links.GetBatchDriveLink(r.Context(), profile.Batch)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteJSON(w, http.StatusOK, driveLinkResponse{
				driveLinkItem: driveLinkItem{Batch: profile.Batch, Title: profile.Batch, IsMine: true},
			})
			return
		}
		if err != nil {
			log.Printf("handler: could not load the drive link of a batch: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, driveLinkResponse{driveLinkItem: newDriveLinkItem(link, profile.Batch)})
	}
}

func (d *DriveLink) CRSave() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := crProfile(w, r, d.links.GetStudentProfile)
		if !ok {
			return
		}
		var request driveLinkRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		link, err := validate.Link("Link", request.Link)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		saved, err := d.links.SaveBatchDriveLink(r.Context(), store.NewDriveLink{
			Batch:     profile.Batch,
			Title:     profile.Batch,
			Link:      link,
			UpdatedBy: profile.UserID,
		})
		if err != nil {
			log.Printf("handler: could not save the drive link of a batch: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, driveLinkResponse{
			driveLinkItem: newDriveLinkItem(saved, profile.Batch),
			Message:       MessageDriveLinkSaved,
		})
	}
}

func (d *DriveLink) CRDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := crProfile(w, r, d.links.GetStudentProfile)
		if !ok {
			return
		}
		err := d.links.DeleteBatchDriveLink(r.Context(), profile.Batch)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageDriveLinkMissing)
		case err != nil:
			log.Printf("handler: could not remove the drive link of a batch: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusNoContent, nil)
		}
	}
}

func (d *DriveLink) StudentList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := studentProfile(w, r, d.links.GetStudentProfile)
		if !ok {
			return
		}
		links, err := d.links.ListBatchDriveLinks(r.Context())
		if err != nil {
			log.Printf("handler: could not list the drive links: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]driveLinkItem, 0, len(links))
		for _, link := range links {
			items = append(items, newDriveLinkItem(link, profile.Batch))
		}
		httpx.WriteJSON(w, http.StatusOK, driveLinksResponse{Items: items})
	}
}

func newDriveLinkItem(link store.DriveLinkRow, ownBatch string) driveLinkItem {
	return driveLinkItem{
		Batch:     link.Batch,
		Title:     link.Title,
		Link:      link.Link,
		UpdatedAt: link.UpdatedAt.Format(time.RFC3339),
		IsMine:    link.Batch == ownBatch,
	}
}
