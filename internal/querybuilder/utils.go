package querybuilder

import (
	"fmt"
	"strings"
)

// backtick escapes the ` characted in strings to make them safe for use in SQL queries as literal values.
func backtick(s string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(backslash(s), "`", "\\`"))
}

func backtickAll(s []string) []string {
	if s == nil {
		return nil
	}
	ret := make([]string, 0)
	for _, p := range s {
		ret = append(ret, backtick(p))
	}
	return ret
}

func quote(s string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(backslash(s), "'", "\\'"))
}

func backslash(s string) string {
	return strings.ReplaceAll(s, "\\", "\\\\")
}

// identifierOrPattern returns a token suitable for use as a database/table identifier
// in GRANT/REVOKE ON clauses.
//
// ClickHouse supports wildcard grants using an asterisk (`*`) as a suffix on database/table
// names (e.g. `db*.*`, `db.table*`). Quoting the pattern (e.g. with backticks) disables
// wildcard matching by turning the pattern into a literal identifier.
func identifierOrPattern(s string) string {
	// Preserve legacy wildcard token.
	if s == "*" {
		return s
	}

	// Only treat a trailing `*` as a wildcard pattern (prefix match), in line with
	// ClickHouse wildcard grant rules.
	if strings.HasSuffix(s, "*") && strings.Count(s, "*") == 1 && !strings.HasPrefix(s, "*") {
		return s
	}
	return backtick(s)
}
