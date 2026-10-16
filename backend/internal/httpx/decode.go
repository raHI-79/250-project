package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const (
	MaxBodyBytes       = 1 << 20
	MessageInvalidBody = "Invalid request body"
)

func DecodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteError(w, http.StatusBadRequest, MessageInvalidBody)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(w, http.StatusBadRequest, MessageInvalidBody)
		return false
	}
	return true
}
