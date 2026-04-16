package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"
)

const requestIDHeader = "X-Request-Id"

type requestIDContextKey struct{}

func observabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !shouldObserveRequest(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		requestID := newRequestID()
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		rec := &statusRecorder{ResponseWriter: w}
		start := time.Now()

		rec.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(rec, r.WithContext(ctx))

		log.Printf(
			"access request_id=%s client=%s method=%s path=%s status=%d bytes=%d duration_ms=%d",
			requestID,
			clientIDFromRequest(r),
			r.Method,
			r.URL.Path,
			rec.StatusCode(),
			rec.BytesWritten(),
			time.Since(start).Milliseconds(),
		)
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func shouldObserveRequest(path string) bool {
	return strings.HasPrefix(path, "/api/") || path == "/healthz" || path == "/readyz"
}

func newRequestID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	return hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) StatusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *statusRecorder) BytesWritten() int {
	return r.bytes
}
