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
