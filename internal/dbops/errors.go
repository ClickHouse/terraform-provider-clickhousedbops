package dbops

import (
	"strings"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
)

// accessEntityAlreadyExistsCode is ClickHouse error code 493
// (ACCESS_ENTITY_ALREADY_EXISTS), returned when creating an access entity
// (settings profile, user, role, ...) that already exists.
const accessEntityAlreadyExistsCode = 493

// findInChain walks the error chain looking for a node matching the given
// predicate. It follows both pingcap/errors wrappers (Cause) and stdlib
// wrappers (Unwrap); pingcap v0.11.4 wrappers do not implement Unwrap, so
// stdlib errors.Is/As alone cannot traverse them.
func findInChain(err error, match func(error) bool) bool {
	for err != nil {
		if match(err) {
			return true
		}

		switch x := err.(type) {
		case interface{ Cause() error }:
			err = x.Cause()
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		default:
			return false
		}
	}

	return false
}

// isAlreadyExistsError reports whether err represents ClickHouse error code
// 493 (ACCESS_ENTITY_ALREADY_EXISTS).
//
// The native protocol client surfaces a typed *clickhouse.Exception carrying
// the code. The HTTP client returns the raw response body as an opaque error,
// so the code has to be matched in the message text.
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	typed := findInChain(err, func(e error) bool {
		ex, ok := e.(*clickhouse.Exception)
		return ok && ex.Code == accessEntityAlreadyExistsCode
	})
	if typed {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "Code: 493.") ||
		strings.Contains(msg, "ACCESS_ENTITY_ALREADY_EXISTS")
}
