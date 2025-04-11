package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

// CreateDatabaseQueryBuilder is an interface to build CREATE DATABASE SQL queries (already interpolated).
type CreateDatabaseQueryBuilder interface {
	QueryBuilder
	WithComment(comment string) CreateDatabaseQueryBuilder
}

type createDatabaseQueryBuilder struct {
	resourceName string
	comment      *string
}

func NewCreateDatabase(resourceName string) CreateDatabaseQueryBuilder {
	return &createDatabaseQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *createDatabaseQueryBuilder) WithComment(comment string) CreateDatabaseQueryBuilder {
	q.comment = &comment
	return q
}

func (q *createDatabaseQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for CREATE and DROP queries")
	}

	tokens := []string{
		"CREATE",
		"DATABASE",
		backtick(q.resourceName),
	}
	if q.comment != nil {
		tokens = append(tokens, fmt.Sprintf("COMMENT %s", quote(*q.comment)))
	}

	return strings.Join(tokens, " ") + ";", nil
}
