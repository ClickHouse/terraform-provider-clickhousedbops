package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

type OrderDirection string

const (
	ASC  OrderDirection = "ASC"
	DESC OrderDirection = "DESC"
)

// SelectQueryBuilder is an interface to build SELECT SQL queries (already interpolated).
type SelectQueryBuilder interface {
	QueryBuilder
	Where(...Where) SelectQueryBuilder
	WithCluster(clusterName *string) SelectQueryBuilder
	LeftArrayJoin(column string, alias string) SelectQueryBuilder
	OrderBy(column Field, order OrderDirection) SelectQueryBuilder
}

type selectQueryBuilder struct {
	tableName       string
	fields          []Field
	where           Where
	clusterName     *string
	arrayJoinColumn string
	arrayJoinAlias  string
	orderBy         Field
	orderDirection  *OrderDirection
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

// LeftArrayJoin unrolls an array or map column into one row per element, keeping
// rows whose column is empty. Used to read Map columns, which the clickhouse
// clients can't decode.
func (q *selectQueryBuilder) LeftArrayJoin(column string, alias string) SelectQueryBuilder {
	q.arrayJoinColumn = column
	q.arrayJoinAlias = alias
	return q
}

func (q *selectQueryBuilder) OrderBy(column Field, order OrderDirection) SelectQueryBuilder {
	q.orderBy = column
	q.orderDirection = &order
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

	if q.arrayJoinColumn != "" {
		if q.arrayJoinAlias == "" {
			return "", errors.New("alias cannot be empty for LEFT ARRAY JOIN")
		}
		tokens = append(tokens, "LEFT ARRAY JOIN", backtick(q.arrayJoinColumn), "AS", backtick(q.arrayJoinAlias))
	}

	// Handle WHERE
	if q.where != nil {
		tokens = append(tokens, "WHERE", q.where.Clause())
	}

	// ORDER BY
	if q.orderBy != nil && q.orderDirection != nil {
		tokens = append(tokens, "ORDER BY", q.orderBy.SQLDef(), string(*q.orderDirection))
	}

	return strings.Join(tokens, " ") + ";", nil
}
