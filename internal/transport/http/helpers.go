package transporthttp

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	stdhttp "net/http"
	"strings"
)

func JSONResp(w stdhttp.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response error: %v", err)
	}
}

func WriteError(w stdhttp.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("encode error response: %v", err)
	}
}

func DecodeJSON(w stdhttp.ResponseWriter, r *stdhttp.Request, limit int64, dst any) bool {
	r.Body = stdhttp.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			WriteError(w, stdhttp.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		WriteError(w, stdhttp.StatusBadRequest, "invalid request body")
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(w, stdhttp.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}

	return true
}
