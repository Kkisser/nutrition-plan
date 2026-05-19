package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type ctxKey int

const ctxRequestID ctxKey = 1

// RequestID возвращает request_id, если он есть в контексте, иначе "".
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxRequestID).(string); ok {
		return v
	}
	return ""
}

// statusRecorder проксирует ResponseWriter и запоминает HTTP-статус.
// http.ResponseWriter сам не выдаёт код, поэтому без обёртки логировать
// статус-ответы невозможно.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// RequestLogger — middleware, который присваивает каждому запросу
// request_id (из заголовка X-Request-ID или сгенерированный) и пишет
// одну структурированную запись на запрос со статусом и длительностью.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := r.Header.Get("X-Request-ID")
			if rid == "" {
				rid = newRequestID()
			}
			ctx := context.WithValue(r.Context(), ctxRequestID, rid)
			w.Header().Set("X-Request-ID", rid)

			sr := &statusRecorder{ResponseWriter: w, status: 200}
			start := time.Now()
			next.ServeHTTP(sr, r.WithContext(ctx))
			dur := time.Since(start)

			level := slog.LevelInfo
			if sr.status >= 500 {
				level = slog.LevelError
			} else if sr.status >= 400 {
				level = slog.LevelWarn
			}
			logger.LogAttrs(ctx, level, "http request",
				slog.String("request_id", rid),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sr.status),
				slog.Duration("duration", dur),
				slog.String("remote", r.RemoteAddr),
			)
		})
	}
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-fallback"
	}
	return hex.EncodeToString(b[:])
}
