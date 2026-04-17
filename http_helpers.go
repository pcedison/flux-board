package main

import (
	"net/http"

	transporthttp "flux-board/internal/transport/http"
)

func jsonResp(w http.ResponseWriter, value any) {
	transporthttp.JSONResp(w, value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	transporthttp.WriteError(w, status, message)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, dst any) bool {
	return transporthttp.DecodeJSON(w, r, limit, dst)
}
