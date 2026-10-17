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
	"strings"
	"time"
)

const (
	MessageMaterialAdded     = "Material added."
	MessageMaterialUnknown   = "No material found with this ID."
	MessageOwnBatchMaterial  = "Pick another batch or share it with your own batch instead."
	MessageBatchForOneBatch  = "Batch is required when the material targets another batch."
	MessageMaterialOwnerOnly = "You can only remove the materials you added."
)

type MaterialStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	CreateMaterial(ctx context.Context, material store.NewMaterial) (store.MaterialRow, error)
	ListMaterials(ctx context.Context, batch string) ([]store.MaterialRow, error)
	ListCreatorMaterials(ctx context.Context, createdBy int64) ([]store.MaterialRow, error)
	ListStudentMaterials(ctx context.Context, batch string) ([]store.MaterialRow, error)
	DeleteMaterial(ctx context.Context, materialID, createdBy int64) error
}

type Material struct {
	materials MaterialStore
}

func NewMaterial(materials MaterialStore) *Material {
	return &Material{materials: materials}
}

type adminMaterialRequest struct {
	Batch string `json:"batch"`
	Title string `json:"title"`
	Link  string `json:"link"`
}

type crMaterialRequest struct {
	Title      string `json:"title"`
	Link       string `json:"link"`
	Visibility string `json:"visibility"`
	Batch      string `json:"batch"`
}

type materialItem struct {
	ID            int64  `json:"id"`
	Batch         string `json:"batch"`
	Title         string `json:"title"`
	Link          string `json:"link"`
	CreatedByName string `json:"created_by_name"`
	CreatedAt     string `json:"created_at"`
}

type materialsResponse struct {
	Items []materialItem `json:"items"`
}

type materialResponse struct {
	materialItem
	Message string `json:"message"`
}

func (m *Material) AdminCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request adminMaterialRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		batch, err := validate.BatchOrAll("Batch", request.Batch)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		title, link, ok := materialContent(w, request.Title, request.Link)
		if !ok {
			return
		}
		m.create(w, r, store.NewMaterial{Batch: batch, Title: title, Link: link, CreatedBy: current.ID})
	}
}

func (m *Material) AdminList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		batch := ""
		if raw := strings.TrimSpace(r.URL.Query().Get("batch")); raw != "" {
			valid, err := validate.BatchOrAll("batch", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			batch = valid
		}
		materials, err := m.materials.ListMaterials(r.Context(), batch)
		m.write(w, materials, err)
	}
}

func (m *Material) CRCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := m.crProfile(w, r)
		if !ok {
			return
		}
		var request crMaterialRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		title, link, ok := materialContent(w, request.Title, request.Link)
		if !ok {
			return
		}
		visibility, err := validate.Visibility("Visibility", request.Visibility)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		batch, ok := materialBatch(w, visibility, request.Batch, profile.Batch)
		if !ok {
			return
		}
		m.create(w, r, store.NewMaterial{Batch: batch, Title: title, Link: link, CreatedBy: profile.UserID})
	}
}

func (m *Material) CRList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := m.crProfile(w, r)
		if !ok {
			return
		}
		materials, err := m.materials.ListCreatorMaterials(r.Context(), profile.UserID)
		m.write(w, materials, err)
	}
}

func (m *Material) CRDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := m.crProfile(w, r)
		if !ok {
			return
		}
		materialID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := m.materials.DeleteMaterial(r.Context(), materialID, profile.UserID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageMaterialUnknown)
		case err != nil:
			log.Printf("handler: could not delete a material: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusNoContent, nil)
		}
	}
}

func (m *Material) StudentList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := studentProfile(w, r, m.materials.GetStudentProfile)
		if !ok {
			return
		}
		materials, err := m.materials.ListStudentMaterials(r.Context(), profile.Batch)
		m.write(w, materials, err)
	}
}

func (m *Material) create(w http.ResponseWriter, r *http.Request, material store.NewMaterial) {
	created, err := m.materials.CreateMaterial(r.Context(), material)
	if err != nil {
		log.Printf("handler: could not create a material: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, materialResponse{
		materialItem: newMaterialItem(created),
		Message:      MessageMaterialAdded,
	})
}

func (m *Material) crProfile(w http.ResponseWriter, r *http.Request) (store.StudentProfile, bool) {
	return crProfile(w, r, m.materials.GetStudentProfile)
}

func (m *Material) write(w http.ResponseWriter, materials []store.MaterialRow, err error) {
	if err != nil {
		log.Printf("handler: could not list the materials: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	items := make([]materialItem, 0, len(materials))
	for _, material := range materials {
		items = append(items, newMaterialItem(material))
	}
	httpx.WriteJSON(w, http.StatusOK, materialsResponse{Items: items})
}

func materialContent(w http.ResponseWriter, rawTitle, rawLink string) (string, string, bool) {
	title, err := validate.Required("Title", rawTitle)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", false
	}
	link, err := validate.Link("Link", rawLink)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", false
	}
	return title, link, true
}

func materialBatch(w http.ResponseWriter, visibility, rawBatch, ownBatch string) (string, bool) {
	switch visibility {
	case validate.VisibilityAll:
		return store.BatchAll, true
	case validate.VisibilityMine:
		return ownBatch, true
	}
	if strings.TrimSpace(rawBatch) == "" {
		httpx.WriteError(w, http.StatusBadRequest, MessageBatchForOneBatch)
		return "", false
	}
	batch, err := validate.Batch("Batch", rawBatch)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", false
	}
	if batch == ownBatch {
		httpx.WriteError(w, http.StatusBadRequest, MessageOwnBatchMaterial)
		return "", false
	}
	return batch, true
}

func newMaterialItem(material store.MaterialRow) materialItem {
	return materialItem{
		ID:            material.ID,
		Batch:         material.Batch,
		Title:         material.Title,
		Link:          material.Link,
		CreatedByName: material.CreatedByName,
		CreatedAt:     material.CreatedAt.Format(time.RFC3339),
	}
}
