package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

const contentTypeJSON = "application/json"

type errorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	if data == nil {
		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(status)
		return
	}
	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("httpx: could not encode response: %v", err)
		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("httpx: could not write response body: %v", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, errorResponse{Error: message})
}
