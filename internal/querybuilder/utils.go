package querybuilder

import (
	"fmt"
	"strings"
)

// escapeChars escapes as ClickHouse writeAnyEscapedString: https://github.com/ClickHouse/ClickHouse/blob/master/src/IO/WriteHelpers.h#L341
var escapeChars = strings.NewReplacer(`\`, `\\`, "\x00", `\0`, "\b", `\b`, "\f", `\f`, "\n", `\n`, "\r", `\r`, "\t", `\t`)

// Escape control characters in the string to use it safely in SQL queries as literal values.
func Escape(s string) string {
	return escapeChars.Replace(s)
}

// QuoteChar quotes the string with a rune `c` and escapes any `c` within the string.
func QuoteChar(s string, c rune) string {
	return fmt.Sprintf("%c%s%c", c, strings.ReplaceAll(Escape(s), string(c), `\`+string(c)), c)
}

// backtick escapes the ` character in strings to make them safe for use in SQL queries as literal values.
func backtick(s string) string {
	return QuoteChar(s, '`')
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
	return QuoteChar(s, '\'')
}

// granteeClause renders a TO clause target: ALL EXCEPT a list, ALL, or an explicit grantee list; empty when nothing is set.
func granteeClause(names []string, all bool, allExcept []string) string {
	if len(allExcept) > 0 {
		return "ALL EXCEPT " + strings.Join(backtickAll(allExcept), ", ")
	}
	if all {
		return "ALL"
	}
	return strings.Join(backtickAll(names), ", ")
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
