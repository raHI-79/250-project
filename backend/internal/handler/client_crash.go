package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
)

const (
	MessageCrashLogged = "Thank you, the problem was reported."
	MessageCrashEmpty  = "Say what went wrong."
	MaxCrashRoute      = 120
	MaxCrashText       = 300
	crashChannelWeb    = "web"
	crashChannelApp    = "app"
)

type crashSink interface {
	WriteChannelEvent(ctx context.Context, event store.SystemEvent) error
}

type clientCrashRequest struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Route   string `json:"route"`
	Client  string `json:"client"`
	Version string `json:"version"`
}

func CrashRoute(value string) string {
	route := strings.TrimSpace(value)
	if cut := strings.IndexAny(route, "?#"); cut >= 0 {
		route = route[:cut]
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	return clipText(route, MaxCrashRoute)
}

func clipText(value string, limit int) string {
	cleaned := strings.Join(strings.Fields(value), " ")
	if len(cleaned) > limit {
		return cleaned[:limit]
	}
	return cleaned
}

func (a *Analytics) Crash() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request clientCrashRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		name := clipText(request.Name, MaxLogFieldLength)
		message := clipText(request.Message, MaxCrashText)
		if name == "" && message == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageCrashEmpty)
			return
		}
		route := CrashRoute(request.Route)
		channel := crashChannelWeb
		if strings.EqualFold(strings.TrimSpace(request.Client), crashChannelApp) {
			channel = crashChannelApp
		}
		version := clipText(request.Version, 20)
		log.Printf("client: the %s %s broke on %s: %s: %s", channel, version, route, name, message)
		if sink, ok := a.views.(crashSink); ok {
			event := store.SystemEvent{
				Kind:    store.KindClientCrash,
				Detail:  route,
				Channel: channel,
				Reason:  name,
				Message: strings.TrimSpace(name + ": " + message),
			}
			if err := sink.WriteChannelEvent(r.Context(), event); err != nil {
				log.Printf("client: could not record the crash: %v", err)
			}
		}
		httpx.WriteJSON(w, http.StatusAccepted, messageResponse{Message: MessageCrashLogged})
	}
}
