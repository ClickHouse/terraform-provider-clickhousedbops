package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

type selectQueryBuilder struct {
	tableName string
	fields    []Field
	options   []Option
}

func NewSelect(fields []Field, from string) QueryBuilder {
	return &selectQueryBuilder{
		fields:    fields,
		tableName: from,
	}
}

func (q *selectQueryBuilder) With(option Option) QueryBuilder {
	q.options = append(q.options, option)
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

	for _, o := range q.options {
		tokens = append(tokens, o.String())
	}

	return strings.Join(tokens, " ") + ";", nil
}
