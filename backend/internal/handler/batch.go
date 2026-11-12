package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	MessageBatchCreated     = "Batch added."
	MessageBatchDuplicate   = "A batch with this name already exists."
	MessageBatchNotFound    = "No batch found with this name."
	MessageBatchSemester    = "Running semester updated."
	MessageBatchDeactivated = "Batch deactivated. Its students are signed out."
	MessageBatchActivated   = "Batch activated."
	MessageBatchDeleted     = "Batch deleted permanently."
	MessageBatchUnchanged   = "The batch is already in that state."
	MessageBatchConfirm     = "Type the batch name exactly to confirm."
	ActionDeleteBatch       = "delete_batch"
	ActionBatchCreated      = "batch_create"
	ActionBatchSemester     = "batch_semester"
	ActionBatchDeactivated  = "batch_deactivate"
	ActionBatchActivated    = "batch_activate"
	ActionBatchDeleted      = "batch_delete"
)

type BatchGate interface {
	BatchOpen(ctx context.Context, name string) (bool, error)
}

type batchGateBox struct {
	gate BatchGate
}

var batchGate atomic.Pointer[batchGateBox]

func UseBatchGate(gate BatchGate) {
	if gate == nil {
		batchGate.Store(nil)
		return
	}
	batchGate.Store(&batchGateBox{gate: gate})
}

func batchClosedMessage(name string) string {
	return name + " is not an active batch."
}

func openBatch(ctx context.Context, label, value string) (string, error) {
	name, err := validate.Batch(label, value)
	if err != nil {
		return "", err
	}
	box := batchGate.Load()
	if box == nil {
		return name, nil
	}
	open, err := box.gate.BatchOpen(ctx, name)
	if err != nil {
		log.Printf("handler: could not check the batch %s: %v", name, err)
		return "", errors.New(middleware.MessageServerError)
	}
	if !open {
		return "", errors.New(batchClosedMessage(name))
	}
	return name, nil
}

type BatchesStore interface {
	BatchGate
	BatchStore
}

type BatchStore interface {
	ListBatchRows(ctx context.Context) ([]store.Batch, error)
	FindBatch(ctx context.Context, name string) (store.Batch, error)
	CreateBatch(ctx context.Context, name, semester string, by int64) (store.Batch, error)
	SetBatchSemester(ctx context.Context, name, semester string, by int64) error
	SetBatchStatus(ctx context.Context, name, status string, by int64) (int, error)
	DeleteBatch(ctx context.Context, name string) (store.BatchRemoval, error)
	BatchWatchers(ctx context.Context, exceptUserID int64) ([]int64, error)
	WriteAudit(ctx context.Context, adminID int64, action, targetSummary string) error
}

type Batches struct {
	cfg     *config.Config
	batches BatchStore
	alerts  Notifier
	events  *Recorder
	asks    *Approvals
}

func NewBatches(cfg *config.Config, batches BatchStore, alerts Notifier) *Batches {
	return &Batches{cfg: cfg, batches: batches, alerts: alerts}
}

func (b *Batches) WithEvents(events *Recorder) *Batches {
	b.events = events
	return b
}

func (b *Batches) WithApprovals(asks *Approvals) *Batches {
	b.asks = asks
	return b
}

type batchItem struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Semester      string   `json:"running_semester"`
	SemesterLabel string   `json:"running_semester_label"`
	Sections      []string `json:"sections"`
	Status        string   `json:"status"`
	Students      int      `json:"students"`
	CreatedBy     string   `json:"created_by"`
	UpdatedBy     string   `json:"updated_by"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type batchesResponse struct {
	Items []batchItem `json:"items"`
}

type batchResponse struct {
	batchItem
	Message string `json:"message"`
}

type batchRequest struct {
	Name     string `json:"name"`
	Semester string `json:"running_semester"`
}

type batchDeleteRequest struct {
	Confirm string `json:"confirm"`
}

func newBatchItem(batch store.Batch) batchItem {
	return batchItem{
		ID:            batch.ID,
		Name:          batch.Name,
		Semester:      batch.RunningSemester,
		SemesterLabel: validate.SemesterLabel(batch.RunningSemester),
		Sections:      batch.Sections,
		Status:        batch.Status,
		Students:      batch.Students,
		CreatedBy:     batch.CreatedBy,
		UpdatedBy:     batch.UpdatedBy,
		CreatedAt:     batch.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     batch.UpdatedAt.Format(time.RFC3339),
	}
}

func (b *Batches) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		batches, err := b.batches.ListBatchRows(r.Context())
		if err != nil {
			log.Printf("handler: could not list the batches: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]batchItem, 0, len(batches))
		for _, batch := range batches {
			items = append(items, newBatchItem(batch))
		}
		httpx.WriteJSON(w, http.StatusOK, batchesResponse{Items: items})
	}
}

func (b *Batches) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request batchRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		name, err := validate.Batch("Batch name", request.Name)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		semester, err := validate.Semester("Running semester", request.Semester)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		created, err := b.batches.CreateBatch(r.Context(), name, semester, current.ID)
		switch {
		case errors.Is(err, store.ErrDuplicateBatch):
			httpx.WriteError(w, http.StatusConflict, MessageBatchDuplicate)
			return
		case err != nil:
			log.Printf("handler: could not create a batch: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		b.record(r, ActionBatchCreated, name,
			fmt.Sprintf("%s added, running semester %s", name, validate.SemesterLabel(semester)))
		httpx.WriteJSON(w, http.StatusCreated, batchResponse{batchItem: newBatchItem(created),
			Message: MessageBatchCreated})
	}
}

func (b *Batches) SetSemester() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, name, ok := b.target(w, r)
		if !ok {
			return
		}
		var request batchRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		semester, err := validate.Semester("Running semester", request.Semester)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		err = b.batches.SetBatchSemester(r.Context(), name, semester, current.ID)
		if !b.changed(w, err, "set the running semester of a batch") {
			return
		}
		b.record(r, ActionBatchSemester, name,
			fmt.Sprintf("%s now runs semester %s", name, validate.SemesterLabel(semester)))
		b.respond(w, r, name, MessageBatchSemester)
	}
}

func (b *Batches) SetStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, name, ok := b.target(w, r)
		if !ok {
			return
		}
		students, err := b.batches.SetBatchStatus(r.Context(), name, status, current.ID)
		if !b.changed(w, err, "change the status of a batch") {
			return
		}
		action, message := ActionBatchActivated, MessageBatchActivated
		summary := fmt.Sprintf("%s activated, %s can sign in again", name, studentCount(students))
		title := name + " was activated"
		if status == store.BatchInactive {
			action, message = ActionBatchDeactivated, MessageBatchDeactivated
			summary = fmt.Sprintf("%s deactivated, %s signed out", name, studentCount(students))
			title = name + " was deactivated"
		}
		b.record(r, action, name, summary)
		b.tell(r, current, title, summary)
		b.respond(w, r, name, message)
	}
}

func (b *Batches) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, name, ok := b.target(w, r)
		if !ok {
			return
		}
		var request batchDeleteRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		batch, err := b.batches.FindBatch(r.Context(), name)
		if !b.changed(w, err, "load a batch") {
			return
		}
		if strings.ToUpper(strings.TrimSpace(request.Confirm)) != batch.Name {
			httpx.WriteError(w, http.StatusBadRequest, MessageBatchConfirm)
			return
		}
		if !current.HasSudo(b.cfg.Now()) {
			httpx.WriteError(w, http.StatusPreconditionRequired, MessageConfirmNeeded)
			return
		}
		summary := fmt.Sprintf("Delete %s for good with its routine, classes, attendance and %s",
			name, studentCount(batch.Students))
		if b.asks != nil && b.asks.Needed(w, r, ActionDeleteBatch, "batch", name, summary, nil) {
			return
		}
		removal, err := b.batches.DeleteBatch(r.Context(), name)
		if !b.changed(w, err, "delete a batch") {
			return
		}
		done := deletedSummary(name, removal)
		b.record(r, ActionBatchDeleted, name, done)
		b.tell(r, current, name+" was deleted", done)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageBatchDeleted})
	}
}

func (b *Batches) RunDelete(ctx context.Context, approval store.Approval) error {
	removal, err := b.batches.DeleteBatch(ctx, approval.TargetID)
	if err != nil {
		return err
	}
	done := deletedSummary(approval.TargetID, removal)
	if err := b.batches.WriteAudit(ctx, approval.RequestedBy, ActionBatchDeleted, done); err != nil {
		log.Printf("handler: could not write an audit entry: %v", err)
	}
	b.notifyWatchers(ctx, 0, approval.TargetID+" was deleted", done)
	return nil
}

func deletedSummary(name string, removal store.BatchRemoval) string {
	return fmt.Sprintf("%s deleted with %s, %s, %s and %s removed", name,
		plural(removal.Routines, "routine entry", "routine entries"),
		plural(removal.Items, "class or exam", "classes and exams"),
		plural(removal.Attendance, "attendance sheet", "attendance sheets"),
		studentCount(removal.Students))
}

func studentCount(total int) string {
	return plural(total, "student", "students")
}

func (b *Batches) target(w http.ResponseWriter, r *http.Request) (middleware.User, string, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return middleware.User{}, "", false
	}
	name, err := validate.Batch("batch", r.PathValue("batch"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return middleware.User{}, "", false
	}
	return current, name, true
}

func (b *Batches) changed(w http.ResponseWriter, err error, doing string) bool {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, MessageBatchNotFound)
	case errors.Is(err, store.ErrBatchUnchanged):
		httpx.WriteError(w, http.StatusConflict, MessageBatchUnchanged)
	case err != nil:
		log.Printf("handler: could not %s: %v", doing, err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
	default:
		return true
	}
	return false
}

func (b *Batches) respond(w http.ResponseWriter, r *http.Request, name, message string) {
	batch, err := b.batches.FindBatch(r.Context(), name)
	if err != nil {
		log.Printf("handler: could not reload the batch %s: %v", name, err)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, batchResponse{batchItem: newBatchItem(batch), Message: message})
}

func (b *Batches) record(r *http.Request, action, name, summary string) {
	if current, ok := middleware.CurrentUser(r.Context()); ok {
		if err := b.batches.WriteAudit(r.Context(), current.ID, action, summary); err != nil {
			log.Printf("handler: could not write an audit entry: %v", err)
		}
	}
	b.events.Record(r, action, "batch", name, summary)
}

func (b *Batches) tell(r *http.Request, current middleware.User, title, body string) {
	b.notifyWatchers(r.Context(), current.ID, title, body+" by "+current.Identifier)
}

func (b *Batches) notifyWatchers(ctx context.Context, exceptUserID int64, title, body string) {
	if b.alerts == nil {
		return
	}
	watchers, err := b.batches.BatchWatchers(ctx, exceptUserID)
	if err != nil {
		log.Printf("handler: could not list the admins to tell about a batch: %v", err)
		return
	}
	for _, userID := range watchers {
		if err := b.alerts.NotifyUser(ctx, userID, notify.TypeDecision, title, body,
			notify.Page(notify.LinkBatches)); err != nil {
			log.Printf("handler: could not tell an admin about a batch: %v", err)
		}
	}
}
