package bthttp

import (
	"net/http"
	"time"

	"github.com/ferux/btcount/internal/btcontext"

	"go.uber.org/zap"
)

const xreqIDHeader = "X-Request-Id"

func middlewareRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xreqid := r.Header.Get(xreqIDHeader)

		ctx := r.Context()
		ctx = btcontext.WithRequestID(ctx, xreqid)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func middlewareLogging(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ctxlog := log.With(zap.String("request_id", btcontext.RequestID(ctx)))
			ctx = btcontext.WithLogger(ctx, ctxlog)
			r = r.WithContext(ctx)

			ctxlog.Debug("incoming request",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)

			start := time.Now()
			ww := &wrappedWriter{ResponseWriter: w}
			next.ServeHTTP(ww, r)

			ctxlog = ctxlog.With(zap.Int("status_code", ww.code), zap.Duration("latency", time.Since(start)))
			switch {
			case ww.code < http.StatusBadRequest: // treat everything less than 400 as ok
				ctxlog.Info("success")
			case ww.code < http.StatusInternalServerError: // everything less than 500 as warn
				ctxlog.Warn("failure")
			default: // everything other as an error
				ctxlog.Error("failure")
			}
		})
	}
}

type wrappedWriter struct {
	http.ResponseWriter

	code int
}

// WriteHeaders saves code.
func (w *wrappedWriter) WriteHeader(code int) {
	w.code = code

	w.ResponseWriter.WriteHeader(code)
}

func middlewareServer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		w.Header().Set(serverHeader, serverName)
	})
}
