package clickhouseclient

import (
	"context"
)

const redactedPlaceholder = "[REDACTED]"

type (
	maskedQueryKey  struct{}
	redactResultKey struct{}
)

// WithMaskedQuery makes the client log masked instead of the query it runs. Callers
// running statements that interpolate secrets pass the query built by
// querybuilder.MaskedQueryBuilder.BuildMasked.
func WithMaskedQuery(ctx context.Context, masked string) context.Context {
	return context.WithValue(ctx, maskedQueryKey{}, masked)
}

// WithRedactedResult keeps the query result out of the logs, for queries reading secrets.
func WithRedactedResult(ctx context.Context) context.Context {
	return context.WithValue(ctx, redactResultKey{}, true)
}

func loggableQuery(ctx context.Context, qry string) string {
	if masked, ok := ctx.Value(maskedQueryKey{}).(string); ok {
		return masked
	}

	return qry
}

func loggableResult(ctx context.Context, result string) string {
	if redacted, ok := ctx.Value(redactResultKey{}).(bool); ok && redacted {
		return redactedPlaceholder
	}

	return result
}
