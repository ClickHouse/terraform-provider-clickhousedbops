package dbops

import (
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouse error code 493 (ACCESS_ENTITY_ALREADY_EXISTS).
const accessEntityAlreadyExistsCode = 493

// findInChain follows both Cause() and Unwrap() chains: pingcap/errors wrappers implement only Cause(), so stdlib errors.Is/As cannot traverse them.
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

// isAlreadyExistsError reports whether err represents ClickHouse error code 493 (ACCESS_ENTITY_ALREADY_EXISTS).
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

	// The HTTP client returns the raw response body as an opaque error, so the code is only matchable as text.
	msg := err.Error()
	return strings.Contains(msg, "Code: 493.") ||
		strings.Contains(msg, "ACCESS_ENTITY_ALREADY_EXISTS")
}
