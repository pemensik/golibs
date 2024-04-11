package slogutil

import (
	"context"
	"io"
	"log/slog"
)

// CloseAndLog is a convenient helper to log errors returned by closer.  The
// point is to not lose information from deferred Close calls.  The error is
// logged with the specified logging level.
//
// Instead of:
//
//	defer f.Close()
//
// You can now write:
//
//	defer slogutil.CloseAndLog(ctx, l, f, slog.LevelDebug)
//
// Note that if closer is nil, it is simply ignored.
func CloseAndLog(ctx context.Context, l *slog.Logger, closer io.Closer, lvl slog.Level) {
	if closer == nil {
		return
	}

	err := closer.Close()
	if err == nil {
		return
	}

	l.Log(ctx, lvl, "deferred closing", KeyError, err)
}

// RecoverAndLog is a deferred helper that recovers from a panic and logs the
// panic value into l along with the stacktrace.
func RecoverAndLog(ctx context.Context, l *slog.Logger) {
	v := recover()
	if v == nil {
		return
	}

	var args []any
	if err, ok := v.(error); ok {
		args = []any{KeyError, err}
	} else {
		args = []any{"value", v}
	}

	l.ErrorContext(ctx, "recovered from panic", args...)
	PrintStack(ctx, l, slog.LevelError)
}
