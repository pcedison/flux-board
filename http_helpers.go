package main

import (
	"net/http"

	transporthttp "flux-board/internal/transport/http"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, dst any) bool {
	return transporthttp.DecodeJSON(w, r, limit, dst)
}
