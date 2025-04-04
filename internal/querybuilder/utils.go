package querybuilder

import (
	"fmt"
	"strings"
)

// escapeBacktick escapes the ` characted in strings to make them safe for use in SQL queries as literal values.
func backtick(s string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(s, "`", "\\`"))
}

func quote(s string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "\\'"))
}
