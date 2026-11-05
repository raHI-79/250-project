package metrics

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/fcm"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/push"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
)

type Sink interface {
	WriteSystemEvent(ctx context.Context, kind string, ok bool, detail string) error
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
	record(ctx, f.sink, store.KindFCM, err == nil, message.Tag)
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
	record(ctx, p.sink, store.KindPush, err == nil, "")
	return status, err
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func Responses(sink Sink) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			watched := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(watched, r)
			if sink == nil {
				return
			}
			ctx := context.WithoutCancel(r.Context())
			switch {
			case watched.status >= 500:
				record(ctx, sink, store.KindClient5x, false, r.Method+" "+r.URL.Path)
			case watched.status >= 400:
				record(ctx, sink, store.KindClient4x, false, r.Method+" "+r.URL.Path)
			}
		})
	}
}
