package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// SelectQueryBuilder is an interface to build SELECT SQL queries (already interpolated).
type SelectQueryBuilder interface {
	QueryBuilder
	Where(...Where) SelectQueryBuilder
}

type selectQueryBuilder struct {
	tableName string
	fields    []Field
	where     Where
}

func NewSelect(fields []Field, from string) SelectQueryBuilder {
	return &selectQueryBuilder{
		fields:    fields,
		tableName: from,
	}
}

func (q *selectQueryBuilder) Where(where ...Where) SelectQueryBuilder {
	q.where = AndWhere(where...)
	return q
}

func (q *selectQueryBuilder) Build() (string, error) {
	if q.tableName == "" {
		return "", errors.New("tableName cannot be empty for SELECT queries")
	}
	if len(q.fields) == 0 {
		return "", errors.New("at least one with is required for SELECT queries")
	}

	fields := make([]string, 0)
	for _, f := range q.fields {
		fields = append(fields, f.SQLDef())
	}

	var from string
	{
		tokens := make([]string, 0)
		for _, s := range strings.Split(q.tableName, ".") {
			tokens = append(tokens, backtick(s))
		}
		from = strings.Join(tokens, ".")
	}

	tokens := []string{
		"SELECT",
		strings.Join(fields, ", "),
		"FROM",
		from,
	}

	// Handle WHERE
	if q.where != nil {
		tokens = append(tokens, "WHERE", q.where.Clause())
	}

	return strings.Join(tokens, " ") + ";", nil
}
