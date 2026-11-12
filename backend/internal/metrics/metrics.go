package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/fcm"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/push"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"time"
)

type Sink interface {
	WriteSystemEvent(ctx context.Context, kind string, ok bool, detail string) error
	WriteChannelEvent(ctx context.Context, event store.SystemEvent) error
}

func recordEvent(ctx context.Context, sink Sink, event store.SystemEvent) {
	if sink == nil {
		return
	}
	event.Detail = clip(event.Detail)
	event.Message = clip(event.Message)
	if err := sink.WriteChannelEvent(ctx, event); err != nil {
		log.Printf("metrics: could not record a %s event: %v", event.Kind, err)
	}
}

func recordChannel(ctx context.Context, sink Sink, kind string, ok bool, detail,
	channel, reason string) {
	recordEvent(ctx, sink, store.SystemEvent{
		Kind:    kind,
		OK:      ok,
		Detail:  detail,
		Channel: channel,
		Reason:  reason,
	})
}

func record(ctx context.Context, sink Sink, kind string, ok bool, detail string) {
	if sink == nil {
		return
	}
	if err := sink.WriteSystemEvent(ctx, kind, ok, detail); err != nil {
		log.Printf("metrics: could not record a %s event: %v", kind, err)
	}
}

type watchedMailer struct {
	inner mailer.Sender
	sink  Sink
}

func Mailer(inner mailer.Sender, sink Sink) mailer.Sender {
	if inner == nil || sink == nil {
		return inner
	}
	return &watchedMailer{inner: inner, sink: sink}
}

func (m *watchedMailer) Configured() bool {
	return m.inner.Configured()
}

func (m *watchedMailer) Send(ctx context.Context, to, subject, body string) error {
	err := m.inner.Send(ctx, to, subject, body)
	record(ctx, m.sink, store.KindEmail, err == nil, subject)
	return err
}

type watchedFCM struct {
	inner fcm.Sender
	sink  Sink
}

func FCM(inner fcm.Sender, sink Sink) fcm.Sender {
	if inner == nil || sink == nil {
		return inner
	}
	return &watchedFCM{inner: inner, sink: sink}
}

func (f *watchedFCM) Configured() bool {
	return f.inner.Configured()
}

func (f *watchedFCM) Send(ctx context.Context, message fcm.Message) error {
	err := f.inner.Send(ctx, message)
	reason := ""
	if err != nil {
		reason = fcm.ReasonOf(err)
	}
	recordChannel(ctx, f.sink, store.KindFCM, err == nil, message.Tag, store.ChannelFCM, reason)
	return err
}

type watchedPush struct {
	inner push.Sender
	sink  Sink
}

func Push(inner push.Sender, sink Sink) push.Sender {
	if inner == nil || sink == nil {
		return inner
	}
	return &watchedPush{inner: inner, sink: sink}
}

func (p *watchedPush) Send(ctx context.Context, subscription store.PushSubscription,
	message []byte) (int, error) {
	status, err := p.inner.Send(ctx, subscription, message)
	reason := push.ReasonOf(status, err)
	recordChannel(ctx, p.sink, store.KindPush, reason == "", "", store.ChannelWebPush, reason)
	return status, err
}

const StatusClientClosed = 499

type statusWriter struct {
	http.ResponseWriter
	ctx    context.Context
	status int
	body   []byte
}

func (w *statusWriter) WriteHeader(status int) {
	if status >= 500 && w.ctx != nil && errors.Is(w.ctx.Err(), context.Canceled) {
		status = StatusClientClosed
	}
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.status >= 400 && len(w.body) < maxDetail {
		w.body = append(w.body, body...)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) message() string {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.body, &payload); err != nil {
		return ""
	}
	return clip(payload.Error)
}

func Responses(sink Sink, counter *Counter) func(http.Handler) http.Handler {
	return responsesStartedAt(time.Now(), sink, counter)
}

func responsesStartedAt(startedAt time.Time, sink Sink,
	counter *Counter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			watched := &statusWriter{ResponseWriter: w, ctx: r.Context()}
			next.ServeHTTP(watched, r)
			probe := IsProbe(r)
			settling := time.Since(startedAt) < SettleWindow
			maintenance := watched.Header().Get(MaintenanceHeader) != ""
			abandoned := errors.Is(r.Context().Err(), context.Canceled)
			serverError := watched.status >= 500 && !probe && !settling && !maintenance && !abandoned
			if !probe {
				counter.Add(serverError)
			}
			if sink == nil {
				return
			}
			ctx := context.WithoutCancel(r.Context())
			switch {
			case serverError:
				recordEvent(ctx, sink, store.SystemEvent{
					Kind:    store.KindClient5x,
					Detail:  r.Method + " " + r.URL.Path,
					Message: watched.message(),
				})
			case watched.status >= 400 && watched.status < 500 && !probe && !abandoned:
				recordEvent(ctx, sink, store.SystemEvent{
					Kind:   store.KindClient4x,
					Detail: r.Method + " " + r.URL.Path,
				})
			}
		})
	}
}
