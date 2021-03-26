package btcontext

import (
	"context"

	"go.uber.org/zap"
)

type logKeyCtx struct{}

// WithLogger wraps context with the logger.
func WithLogger(ctx context.Context, log *zap.Logger) (newctx context.Context) {
	return context.WithValue(ctx, logKeyCtx{}, log)
}

// Logger gets logger from context if exists or return nop logger.
func Logger(ctx context.Context) (log *zap.Logger) {
	var ok bool
	log, ok = ctx.Value(logKeyCtx{}).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}

	return log
}

type xreqidKeyCtx struct{}

// WithRequestID wraps context with the request id.
func WithRequestID(ctx context.Context, xreqid string) (newctx context.Context) {
	return context.WithValue(ctx, xreqidKeyCtx{}, xreqid)
}

// RequestID gets request id from context.
func RequestID(ctx context.Context) (xreqid string) {
	var ok bool
	xreqid, ok = ctx.Value(xreqidKeyCtx{}).(string)
	if !ok {
		return ""
	}

	return xreqid
}
