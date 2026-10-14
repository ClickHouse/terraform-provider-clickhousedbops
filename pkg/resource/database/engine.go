package database

import (
	"regexp"
	"strings"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

var (
	engineNamePattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	stringLiteralPattern = `'(?:[^'\\]|\\.)*'`
	// hiddenLiteral is how ClickHouse reports a secret engine argument or setting.
	hiddenLiteral = "'" + querybuilder.HiddenValue + "'"
	// secretExpressionPattern matches the configured expressions ClickHouse may hide.
	secretExpressionPattern = regexp.MustCompile(`^(?:` + stringLiteralPattern + `|` + querybuilder.EngineParameterPattern.String() + `)$`)
)

// engineClause is a database engine clause as reported by system.databases.engine_full.
type engineClause struct {
	name      string
	arguments []string
	settings  map[string]string
}

// parseEngineFull splits an engine_full value such as
// `Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}')` or
// `DataLakeCatalog('https://catalog.example.test') SETTINGS catalog_type = 'rest'` into its
// parts. It returns false for any value it does not recognise.
func parseEngineFull(engineFull string) (engineClause, bool) {
	rest := strings.TrimSpace(engineFull)
	nameEnd := strings.IndexAny(rest, "( ")
	if nameEnd == -1 {
		nameEnd = len(rest)
	}

	clause := engineClause{name: rest[:nameEnd]}
	if !engineNamePattern.MatchString(clause.name) {
		return engineClause{}, false
	}
	rest = rest[nameEnd:]

	if strings.HasPrefix(rest, "(") {
		arguments, end, ok := splitExpressions(rest[1:], ')')
		if !ok {
			return engineClause{}, false
		}
		if len(arguments) > 1 || arguments[0] != "" {
			clause.arguments = arguments
		}
		rest = rest[1+end+1:]
	}

	rest = strings.TrimSpace(rest)
	if rest == "" {
		return clause, true
	}

	settings, found := strings.CutPrefix(rest, "SETTINGS ")
	if !found {
		return engineClause{}, false
	}
	pairs, _, ok := splitExpressions(settings, 0)
	if !ok {
		return engineClause{}, false
	}
	clause.settings = make(map[string]string, len(pairs))
	for _, pair := range pairs {
		name, value, found := strings.Cut(pair, "=")
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		if !found || !engineNamePattern.MatchString(name) || value == "" {
			return engineClause{}, false
		}
		clause.settings[name] = value
	}

	return clause, true
}

// splitExpressions splits s on commas outside quotes and brackets, up to the first unmatched
// stop byte, or to the end of s when stop is 0. It returns the trimmed expressions and the
// index of the stop byte.
func splitExpressions(s string, stop byte) ([]string, int, bool) {
	var (
		expressions []string
		depth       int
		quote       byte
		start       int
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case c == '(' || c == '[' || c == '{':
			depth++
		case depth == 0 && stop != 0 && c == stop:
			return append(expressions, strings.TrimSpace(s[start:i])), i, true
		case c == ')' || c == ']' || c == '}':
			depth--
			if depth < 0 {
				return nil, 0, false
			}
		case depth == 0 && c == ',':
			expressions = append(expressions, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	if stop != 0 || quote != 0 || depth != 0 {
		return nil, 0, false
	}

	return append(expressions, strings.TrimSpace(s[start:])), len(s), true
}

func (c engineClause) matchesArguments(configured []string) bool {
	if len(configured) != len(c.arguments) {
		return false
	}
	for i := range configured {
		if !engineExpressionMatches(configured[i], c.arguments[i]) {
			return false
		}
	}

	return true
}

func (c engineClause) matchesSettings(configured map[string]string) bool {
	if len(configured) != len(c.settings) {
		return false
	}
	for name, value := range configured {
		actual, ok := c.settings[name]
		if !ok || !engineExpressionMatches(value, actual) {
			return false
		}
	}

	return true
}

// engineExpressionMatches reports whether a configured engine argument or setting value is
// the expression ClickHouse reports. ClickHouse formats expressions canonically, so the
// comparison is textual. A write-only parameter placeholder matches any string literal,
// and a value hidden by ClickHouse matches any string literal or placeholder.
func engineExpressionMatches(configured, actual string) bool {
	configured = strings.TrimSpace(configured)
	if actual == hiddenLiteral && secretExpressionPattern.MatchString(configured) {
		return true
	}

	var pattern strings.Builder
	pattern.WriteString("^")
	last := 0
	for _, placeholder := range querybuilder.EngineParameterPattern.FindAllStringIndex(configured, -1) {
		pattern.WriteString(regexp.QuoteMeta(configured[last:placeholder[0]]))
		pattern.WriteString(stringLiteralPattern)
		last = placeholder[1]
	}
	pattern.WriteString(regexp.QuoteMeta(configured[last:]))
	pattern.WriteString("$")

	return regexp.MustCompile(pattern.String()).MatchString(actual)
}
