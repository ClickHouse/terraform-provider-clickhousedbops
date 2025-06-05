package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

// SelectQueryBuilder is an interface to build SELECT SQL queries (already interpolated).
type SelectQueryBuilder interface {
	QueryBuilder
	Where(...Where) SelectQueryBuilder
	WithCluster(clusterName *string) SelectQueryBuilder
}

type selectQueryBuilder struct {
	tableName   string
	fields      []Field
	where       Where
	clusterName *string
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

func (q *selectQueryBuilder) WithCluster(clusterName *string) SelectQueryBuilder {
	q.clusterName = clusterName
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
		tableName := strings.Join(tokens, ".")

		if q.clusterName != nil {
			from = fmt.Sprintf("cluster(%s, %s)", quote(*q.clusterName), tableName)
		} else {
			from = tableName
		}
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
