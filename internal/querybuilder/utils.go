package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
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

// privilegeTarget renders one of ClickHouse's two ON target families:
// database/table targets and global-with-parameter targets. The latter includes
// users, named collections, table engines, sources, and source regexp filters.
func privilegeTarget(database, table, accessObject, accessObjectFilter *string, parameterized bool) (string, error) {
	if accessObjectFilter != nil && accessObject == nil {
		return "", errors.New("AccessObjectFilter requires AccessObject")
	}
	if accessObject != nil && (database != nil || table != nil) {
		return "", errors.New("AccessObject cannot be combined with Database or Table")
	}
	if parameterized && (database != nil || table != nil) {
		return "", errors.New("a parameterized privilege target cannot use Database or Table")
	}

	switch {
	case accessObjectFilter != nil:
		return fmt.Sprintf("%s(%s)", identifierOrPattern(*accessObject), quote(*accessObjectFilter)), nil
	case accessObject != nil:
		return identifierOrPattern(*accessObject), nil
	case parameterized:
		return "*", nil
	case database != nil && table != nil:
		return fmt.Sprintf("%s.%s", identifierOrPattern(*database), identifierOrPattern(*table)), nil
	case database != nil:
		return fmt.Sprintf("%s.*", identifierOrPattern(*database)), nil
	default:
		return "*.*", nil
	}
}
