package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"time"
)

const (
	MessagePollCreated    = "Poll created."
	MessagePollClosed     = "This poll is closed."
	MessagePollUnknown    = "No poll found with this ID."
	MessageAlreadyVoted   = "You have already voted in this poll."
	MessageOptionUnknown  = "This option does not belong to this poll."
	MessageVoteSaved      = "Your vote is saved."
	MessageTooManyChoices = "This poll allows fewer choices than that."
	MessagePickAnOption   = "Pick at least one option."
	MessagePollOpen       = "This poll is already open."
	MessagePollHasVotes   = "This poll already has votes, so it can no longer be edited."
	MessagePollEdited     = "Poll updated."
	MessagePollReopened   = "Poll reopened."
	MessagePollDeleted    = "Poll deleted."
	NotificationTypePoll  = "poll"
)

type PollStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	CreatePoll(ctx context.Context, poll store.NewPoll) (store.PollRow, error)
	ClosePoll(ctx context.Context, pollID, createdBy int64, now time.Time) error
	ReopenPoll(ctx context.Context, pollID, createdBy int64) error
	DeletePoll(ctx context.Context, pollID, createdBy int64) error
	EditPoll(ctx context.Context, pollID, createdBy int64, question string, maxChoices int,
		options []string) error
	ListCreatorPolls(ctx context.Context, viewerUserID, createdBy int64) ([]store.PollRow, error)
	ListStudentPolls(ctx context.Context, viewerUserID int64, batch, section string) ([]store.PollRow, error)
	VoteInPoll(ctx context.Context, pollID int64, optionIDs []int64, userID int64, batch, section string) error
}

type Poll struct {
	cfg    *config.Config
	polls  PollStore
	alerts BroadcastNotifier
}

func NewPoll(cfg *config.Config, polls PollStore, alerts BroadcastNotifier) *Poll {
	return &Poll{cfg: cfg, polls: polls, alerts: alerts}
}

type createPollRequest struct {
	Question   string   `json:"question"`
	Options    []string `json:"options"`
	Scope      string   `json:"scope"`
	MaxChoices int      `json:"max_choices"`
}

type voteRequest struct {
	OptionIDs []int64 `json:"option_ids"`
}

type pollOptionItem struct {
	ID    int64  `json:"id"`
	Text  string `json:"text"`
	Votes int    `json:"votes"`
}

type pollItem struct {
	ID            int64            `json:"id"`
	Question      string           `json:"question"`
	Scope         string           `json:"scope"`
	Batch         string           `json:"batch"`
	Section       string           `json:"section"`
	IsClosed      bool             `json:"is_closed"`
	MaxChoices    int              `json:"max_choices"`
	CreatedByName string           `json:"created_by_name"`
	CreatedAt     string           `json:"created_at"`
	EditedAt      string           `json:"edited_at,omitempty"`
	Options       []pollOptionItem `json:"options"`
	TotalVoters   int              `json:"total_voters"`
	MyOptionIDs   []int64          `json:"my_option_ids"`
}

type pollsResponse struct {
	Items []pollItem `json:"items"`
}

type pollResponse struct {
	pollItem
	Message string `json:"message"`
}

func (p *Poll) CRCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := p.crProfile(w, r)
		if !ok {
			return
		}
		var request createPollRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		question, err := validate.Question("Question", request.Question)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		options, err := validate.PollOptions("Options", request.Options)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		scope, err := validate.PollScope("Scope", request.Scope)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		maxChoices, err := validate.PollChoices("Choices", request.MaxChoices, len(options))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		batch, section := pollAudience(scope, profile)
		created, err := p.polls.CreatePoll(r.Context(), store.NewPoll{
			Question:   question,
			Scope:      scope,
			Batch:      batch,
			Section:    section,
			Options:    options,
			MaxChoices: maxChoices,
			CreatedBy:  profile.UserID,
		})
		if err != nil {
			log.Printf("handler: could not create a poll: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		audienceBatch, audienceSection := notifyAudience(created)
		err = p.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, audienceBatch, audienceSection,
			NotificationTypePoll, "New poll", created.Question, notify.Ref(notify.LinkPoll, created.ID))
		if err != nil {
			log.Printf("handler: could not notify the students of a poll: %v", err)
		}
		httpx.WriteJSON(w, http.StatusCreated, pollResponse{pollItem: newPollItem(created), Message: MessagePollCreated})
	}
}

func (p *Poll) CRClose() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := p.crProfile(w, r)
		if !ok {
			return
		}
		pollID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := p.polls.ClosePoll(r.Context(), pollID, profile.UserID, p.cfg.Now())
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessagePollUnknown)
		case errors.Is(err, store.ErrPollClosed):
			httpx.WriteError(w, http.StatusConflict, MessagePollClosed)
		case err != nil:
			log.Printf("handler: could not close a poll: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "Poll closed."})
		}
	}
}

func (p *Poll) CRReopen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := p.crProfile(w, r)
		if !ok {
			return
		}
		pollID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := p.polls.ReopenPoll(r.Context(), pollID, profile.UserID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessagePollUnknown)
		case errors.Is(err, store.ErrPollOpen):
			httpx.WriteError(w, http.StatusConflict, MessagePollOpen)
		case err != nil:
			log.Printf("handler: could not reopen a poll: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePollReopened})
		}
	}
}

func (p *Poll) CRDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := p.crProfile(w, r)
		if !ok {
			return
		}
		pollID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := p.polls.DeletePoll(r.Context(), pollID, profile.UserID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessagePollUnknown)
		case err != nil:
			log.Printf("handler: could not delete a poll: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePollDeleted})
		}
	}
}

func (p *Poll) CREdit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := p.crProfile(w, r)
		if !ok {
			return
		}
		pollID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request createPollRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		question, err := validate.Question("Question", request.Question)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		options, err := validate.PollOptions("Options", request.Options)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		maxChoices, err := validate.PollChoices("Choices", request.MaxChoices, len(options))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		err = p.polls.EditPoll(r.Context(), pollID, profile.UserID, question, maxChoices, options)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessagePollUnknown)
		case errors.Is(err, store.ErrPollHasVotes):
			httpx.WriteError(w, http.StatusConflict, MessagePollHasVotes)
		case err != nil:
			log.Printf("handler: could not edit a poll: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePollEdited})
		}
	}
}

func (p *Poll) CRList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := p.crProfile(w, r)
		if !ok {
			return
		}
		polls, err := p.polls.ListCreatorPolls(r.Context(), profile.UserID, profile.UserID)
		p.write(w, polls, err)
	}
}

func (p *Poll) StudentList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := studentProfile(w, r, p.polls.GetStudentProfile)
		if !ok {
			return
		}
		polls, err := p.polls.ListStudentPolls(r.Context(), profile.UserID, profile.Batch, profile.Section)
		p.write(w, polls, err)
	}
}

func (p *Poll) StudentVote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := studentProfile(w, r, p.polls.GetStudentProfile)
		if !ok {
			return
		}
		pollID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request voteRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		choices, err := distinctChoices(request.OptionIDs)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		err = p.polls.VoteInPoll(r.Context(), pollID, choices, profile.UserID, profile.Batch, profile.Section)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessagePollUnknown)
		case errors.Is(err, store.ErrPollClosed):
			httpx.WriteError(w, http.StatusConflict, MessagePollClosed)
		case errors.Is(err, store.ErrTooManyChoices):
			httpx.WriteError(w, http.StatusBadRequest, MessageTooManyChoices)
		case errors.Is(err, store.ErrOptionNotInPoll):
			httpx.WriteError(w, http.StatusBadRequest, MessageOptionUnknown)
		case errors.Is(err, store.ErrAlreadyVoted):
			httpx.WriteError(w, http.StatusConflict, MessageAlreadyVoted)
		case err != nil:
			log.Printf("handler: could not save a vote: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusCreated, map[string]string{"message": MessageVoteSaved})
		}
	}
}

func (p *Poll) crProfile(w http.ResponseWriter, r *http.Request) (store.StudentProfile, bool) {
	return crProfile(w, r, p.polls.GetStudentProfile)
}

func (p *Poll) write(w http.ResponseWriter, polls []store.PollRow, err error) {
	if err != nil {
		log.Printf("handler: could not list the polls: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	items := make([]pollItem, 0, len(polls))
	for _, poll := range polls {
		items = append(items, newPollItem(poll))
	}
	httpx.WriteJSON(w, http.StatusOK, pollsResponse{Items: items})
}

func newPollItem(poll store.PollRow) pollItem {
	options := make([]pollOptionItem, 0, len(poll.Options))
	for _, option := range poll.Options {
		options = append(options, pollOptionItem{ID: option.ID, Text: option.Text, Votes: option.Votes})
	}
	mine := poll.MyOptionIDs
	if mine == nil {
		mine = make([]int64, 0)
	}
	return pollItem{
		ID:            poll.ID,
		Question:      poll.Question,
		Scope:         poll.Scope,
		Batch:         poll.Batch,
		Section:       poll.Section,
		IsClosed:      poll.Status == store.PollClosed,
		MaxChoices:    poll.MaxChoices,
		CreatedByName: poll.CreatedByName,
		CreatedAt:     poll.CreatedAt.Format(time.RFC3339),
		EditedAt:      optionalStamp(poll.EditedAt),
		Options:       options,
		TotalVoters:   poll.TotalVoters,
		MyOptionIDs:   mine,
	}
}

func pollAudience(scope string, profile store.StudentProfile) (string, string) {
	switch scope {
	case validate.PollScopeAll:
		return validate.BatchAll, validate.SectionAll
	case validate.PollScopeBatch:
		return profile.Batch, validate.SectionAll
	default:
		return profile.Batch, profile.Section
	}
}

func notifyAudience(poll store.PollRow) (string, string) {
	switch poll.Scope {
	case validate.PollScopeAll:
		return "", ""
	case validate.PollScopeBatch:
		return poll.Batch, ""
	default:
		return poll.Batch, poll.Section
	}
}

func distinctChoices(raw []int64) ([]int64, error) {
	seen := make(map[int64]bool, len(raw))
	choices := make([]int64, 0, len(raw))
	for _, id := range raw {
		if id <= 0 {
			return nil, errors.New(MessageOptionUnknown)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		choices = append(choices, id)
	}
	if len(choices) == 0 {
		return nil, errors.New(MessagePickAnOption)
	}
	return choices, nil
}
