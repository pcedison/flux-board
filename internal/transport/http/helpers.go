package transporthttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	stdhttp "net/http"
	"strings"
)

func JSONResp(w stdhttp.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Default().Error("encode response error", slog.Any("err", err))
	}
}

func WriteError(w stdhttp.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Default().Error("encode error response", slog.Any("err", err))
	}
}

func DecodeJSON(w stdhttp.ResponseWriter, r *stdhttp.Request, limit int64, dst any) bool {
	r.Body = stdhttp.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		logger := LoggerFromContext(r.Context())
		if strings.Contains(err.Error(), "http: request body too large") {
			WriteError(w, stdhttp.StatusRequestEntityTooLarge, "request body too large")
			logger.Warn("request body too large", slog.Any("err", err))
			return false
		}
		WriteError(w, stdhttp.StatusBadRequest, "invalid request body")
		logger.Warn("decode request body failed", slog.Any("err", err))
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(w, stdhttp.StatusBadRequest, "request body must contain a single JSON object")
		LoggerFromContext(r.Context()).Warn("reject trailing JSON payload", slog.Any("err", err))
		return false
	}

	return true
}
