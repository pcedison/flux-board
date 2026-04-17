package transporthttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	stdhttp "net/http"
	"strings"
	"time"
)

const RequestIDHeader = "X-Request-Id"

type requestIDContextKey struct{}

func ObservabilityMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if !ShouldObserveRequest(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		requestID := newRequestID()
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		rec := &statusRecorder{ResponseWriter: w}
		start := time.Now()

		rec.Header().Set(RequestIDHeader, requestID)
		next.ServeHTTP(rec, r.WithContext(ctx))

		log.Printf(
			"access request_id=%s client=%s method=%s path=%s status=%d bytes=%d duration_ms=%d",
			requestID,
			ClientIDFromRequest(r),
			r.Method,
			r.URL.Path,
			rec.StatusCode(),
			rec.BytesWritten(),
			time.Since(start).Milliseconds(),
		)
	})
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func ShouldObserveRequest(path string) bool {
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
	stdhttp.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = stdhttp.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) StatusCode() int {
	if r.status == 0 {
		return stdhttp.StatusOK
	}
	return r.status
}

func (r *statusRecorder) BytesWritten() int {
	return r.bytes
}
