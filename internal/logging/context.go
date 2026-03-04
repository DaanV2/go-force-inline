package logging

import (
	"context"

	"github.com/charmbracelet/log"
)

type loggingContextKey struct{}

// From returns the possible injected logger in the context
func From(ctx context.Context) *log.Logger {
	if ctx != nil {
		v := ctx.Value(loggingContextKey{})
		if v != nil {
			logger, ok := v.(*log.Logger)
			if ok {
				return logger
			}
		}
	}

	return log.Default()
}

// FromPrefix returns the possible injected logger in the context, adding the given prefix
func FromPrefix(ctx context.Context, prefix string) *log.Logger {
	return From(ctx).WithPrefix(prefix)
}

// Context returns a new context with the given logger attached
func Context(ctx context.Context, logger *log.Logger) context.Context {
	if logger == nil {
		return ctx
	}

	return context.WithValue(ctx, loggingContextKey{}, logger)
}
