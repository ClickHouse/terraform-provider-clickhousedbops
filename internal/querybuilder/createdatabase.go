package querybuilder

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pingcap/errors"
)

// CreateDatabaseQueryBuilder is an interface to build CREATE DATABASE SQL queries (already interpolated).
type CreateDatabaseQueryBuilder interface {
	QueryBuilder
	WithComment(comment string) CreateDatabaseQueryBuilder
	WithCluster(clusterName *string) CreateDatabaseQueryBuilder
	WithEngine(name string, arguments []string, settings map[string]string) CreateDatabaseQueryBuilder
	WithParameters(parameters map[string]string) CreateDatabaseQueryBuilder
	RedactedQuery() (string, error)
}

type createDatabaseQueryBuilder struct {
	databaseName    string
	comment         *string
	clusterName     *string
	engineName      *string
	engineArguments []string
	engineSettings  map[string]string
	queryParameters map[string]string
}

func NewCreateDatabase(name string) CreateDatabaseQueryBuilder {
	return &createDatabaseQueryBuilder{
		databaseName: name,
	}
}

func (q *createDatabaseQueryBuilder) WithComment(comment string) CreateDatabaseQueryBuilder {
	q.comment = &comment
	return q
}

func (q *createDatabaseQueryBuilder) WithCluster(clusterName *string) CreateDatabaseQueryBuilder {
	q.clusterName = clusterName
	return q
}

// WithEngine configures a database engine. Arguments and setting values are ClickHouse
// expressions rather than string values: callers must include any required quoting. This
// keeps the builder compatible with engines which accept identifiers, arrays, numbers, or
// functions in addition to strings.
func (q *createDatabaseQueryBuilder) WithEngine(name string, arguments []string, settings map[string]string) CreateDatabaseQueryBuilder {
	q.engineName = &name
	q.engineArguments = arguments
	q.engineSettings = settings
	return q
}

// WithParameters attaches write-only string parameters referenced by engine arguments or
// settings (for example, {catalog_credential:String}). Build replaces placeholders with
// quoted values; RedactedQuery replaces them with a fixed marker for safe provider logging.
func (q *createDatabaseQueryBuilder) WithParameters(parameters map[string]string) CreateDatabaseQueryBuilder {
	q.queryParameters = parameters
	return q
}

func (q *createDatabaseQueryBuilder) RedactedQuery() (string, error) {
	query, err := q.build()
	if err != nil {
		return "", err
	}
	return q.substituteParameters(query, true)
}

func (q *createDatabaseQueryBuilder) Build() (string, error) {
	query, err := q.build()
	if err != nil {
		return "", err
	}
	return q.substituteParameters(query, false)
}

func (q *createDatabaseQueryBuilder) build() (string, error) {
	if q.databaseName == "" {
		return "", errors.New("databaseName cannot be empty for CREATE DATABASE queries")
	}
	if q.engineName != nil && *q.engineName == "" {
		return "", errors.New("engineName cannot be empty for CREATE DATABASE queries")
	}

	tokens := []string{
		"CREATE",
		"DATABASE",
		backtick(q.databaseName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}
	if q.engineName != nil {
		engine := backtick(*q.engineName)
		if len(q.engineArguments) > 0 {
			engine += fmt.Sprintf("(%s)", strings.Join(q.engineArguments, ", "))
		}
		tokens = append(tokens, "ENGINE", "=", engine)

		if len(q.engineSettings) > 0 {
			keys := make([]string, 0, len(q.engineSettings))
			for key := range q.engineSettings {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			settings := make([]string, 0, len(keys))
			for _, key := range keys {
				settings = append(settings, fmt.Sprintf("%s = %s", backtick(key), q.engineSettings[key]))
			}
			tokens = append(tokens, "SETTINGS", strings.Join(settings, ", "))
		}
	}
	if q.comment != nil {
		tokens = append(tokens, "COMMENT", quote(*q.comment))
	}

	return strings.Join(tokens, " ") + ";", nil
}

var engineParameterPattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*):String\}`)

func (q *createDatabaseQueryBuilder) substituteParameters(query string, redact bool) (string, error) {
	matches := engineParameterPattern.FindAllStringSubmatch(query, -1)
	referenced := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		name := match[1]
		_, ok := q.queryParameters[name]
		if !ok {
			return "", fmt.Errorf("engine parameter %q is referenced but not configured", name)
		}
		referenced[name] = struct{}{}
	}
	for name := range q.queryParameters {
		if _, ok := referenced[name]; !ok {
			return "", fmt.Errorf("engine parameter %q is configured but not referenced", name)
		}
	}

	// Replace in one pass so placeholder-like text inside a parameter value cannot
	// be interpreted as another parameter.
	return engineParameterPattern.ReplaceAllStringFunc(query, func(placeholder string) string {
		name := engineParameterPattern.FindStringSubmatch(placeholder)[1]
		value := q.queryParameters[name]
		if redact {
			value = "[REDACTED]"
		}
		return quote(value)
	}), nil
}
