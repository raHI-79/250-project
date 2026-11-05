package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net"
	"net/http"
	"strings"
)

const maxUserAgent = 300

type EventSink interface {
	WriteEvent(ctx context.Context, event store.AuditEvent) error
}

type Recorder struct {
	sink EventSink
	salt string
}

func NewRecorder(cfg *config.Config, sink EventSink) *Recorder {
	salt := ""
	if cfg != nil {
		salt = cfg.AnalyticsSalt
	}
	return &Recorder{sink: sink, salt: salt}
}

func (rec *Recorder) Record(r *http.Request, action, targetType, targetID, summary string) {
	if rec == nil || rec.sink == nil {
		return
	}
	current, ok := middleware.CurrentUser(r.Context())
	actorID := int64(0)
	role := ""
	label := ""
	if ok {
		actorID = current.ID
		role = current.Role
		label = current.Identifier
	}
	rec.write(r, actorID, role, label, action, targetType, targetID, summary)
}

func (rec *Recorder) RecordAs(r *http.Request, actorID int64, role, label, action,
	targetType, targetID, summary string) {
	if rec == nil || rec.sink == nil {
		return
	}
	rec.write(r, actorID, role, label, action, targetType, targetID, summary)
}

func (rec *Recorder) write(r *http.Request, actorID int64, role, label, action,
	targetType, targetID, summary string) {
	event := store.AuditEvent{
		ActorID:    actorID,
		ActorRole:  role,
		ActorLabel: label,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Summary:    summary,
		IPHash:     HashAddress(rec.salt, RequestAddress(r)),
		UserAgent:  requestAgent(r),
		Client:     RequestClient(r),
	}
	if err := rec.sink.WriteEvent(r.Context(), event); err != nil {
		log.Printf("handler: could not record the %s event: %v", action, err)
	}
}

func RequestClient(r *http.Request) string {
	if middleware.FromAndroidApp(r) {
		return store.ClientAndroid
	}
	return store.ClientWeb
}

func RequestAddress(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func HashAddress(salt, address string) string {
	if address == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(salt + "|" + address))
	return hex.EncodeToString(sum[:])
}

func requestAgent(r *http.Request) string {
	agent := strings.TrimSpace(r.UserAgent())
	if len(agent) > maxUserAgent {
		return agent[:maxUserAgent]
	}
	return agent
}
