package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("response: %v", err)
	}
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "send one JSON object with the documented fields"})
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		respond(w, http.StatusBadRequest, map[string]string{"error": "send exactly one JSON object"})
		return false
	}
	return true
}
