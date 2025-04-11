package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// CreateDatabaseQueryBuilder is an interface to build CREATE DATABASE SQL queries (already interpolated).
type CreateDatabaseQueryBuilder interface {
	QueryBuilder
	WithComment(comment string) CreateDatabaseQueryBuilder
}

type createDatabaseQueryBuilder struct {
	databaseName string
	comment      *string
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

func (q *createDatabaseQueryBuilder) Build() (string, error) {
	if q.databaseName == "" {
		return "", errors.New("databaseName cannot be empty for CREATE DATABASE queries")
	}

	tokens := []string{
		"CREATE",
		"DATABASE",
		backtick(q.databaseName),
	}
	if q.comment != nil {
		tokens = append(tokens, "COMMENT", quote(*q.comment))
	}

	return strings.Join(tokens, " ") + ";", nil
}
