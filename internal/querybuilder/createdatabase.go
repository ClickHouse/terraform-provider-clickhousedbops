package querybuilder

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/pingcap/errors"
)

// CreateDatabaseQueryBuilder is an interface to build CREATE DATABASE SQL queries (already interpolated).
type CreateDatabaseQueryBuilder interface {
	QueryBuilder
	WithComment(comment string) CreateDatabaseQueryBuilder
	WithCluster(clusterName *string) CreateDatabaseQueryBuilder
	WithEngine(name string, arguments []string, settings map[string]string) CreateDatabaseQueryBuilder
}

type createDatabaseQueryBuilder struct {
	databaseName    string
	comment         *string
	clusterName     *string
	engineName      *string
	engineArguments []string
	engineSettings  map[string]string
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

// WithEngine takes arguments and setting values as raw SQL expressions, callers quote string literals.
func (q *createDatabaseQueryBuilder) WithEngine(name string, arguments []string, settings map[string]string) CreateDatabaseQueryBuilder {
	q.engineName = &name
	q.engineArguments = arguments
	q.engineSettings = settings
	return q
}

func (q *createDatabaseQueryBuilder) Build() (string, error) {
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
		engine := *q.engineName
		if len(q.engineArguments) > 0 {
			engine += fmt.Sprintf("(%s)", strings.Join(q.engineArguments, ", "))
		}
		tokens = append(tokens, "ENGINE", "=", engine)

		if len(q.engineSettings) > 0 {
			settings := make([]string, 0, len(q.engineSettings))
			for _, key := range slices.Sorted(maps.Keys(q.engineSettings)) {
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
