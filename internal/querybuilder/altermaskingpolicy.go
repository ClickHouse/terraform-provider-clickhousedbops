package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

// AlterMaskingPolicyQueryBuilder builds ALTER MASKING POLICY statements.
type AlterMaskingPolicyQueryBuilder interface {
	QueryBuilder
	RenameTo(string) AlterMaskingPolicyQueryBuilder
	WithWhere(*string) AlterMaskingPolicyQueryBuilder
	GranteeNames([]string) AlterMaskingPolicyQueryBuilder
	GranteeAll(bool) AlterMaskingPolicyQueryBuilder
	GranteeAllExcept([]string) AlterMaskingPolicyQueryBuilder
	WithPriority(*int64) AlterMaskingPolicyQueryBuilder
}

type alterMaskingPolicyQueryBuilder struct {
	name             string
	database         string
	table            string
	masks            []ColumnMask
	renameTo         *string
	where            *string
	granteeNames     []string
	granteeAll       bool
	granteeAllExcept []string
	priority         *int64
}

func NewAlterMaskingPolicy(name string, database string, table string, masks []ColumnMask) AlterMaskingPolicyQueryBuilder {
	return &alterMaskingPolicyQueryBuilder{
		name:     name,
		database: database,
		table:    table,
		masks:    masks,
	}
}

func (q *alterMaskingPolicyQueryBuilder) RenameTo(newName string) AlterMaskingPolicyQueryBuilder {
	q.renameTo = &newName
	return q
}

func (q *alterMaskingPolicyQueryBuilder) WithWhere(where *string) AlterMaskingPolicyQueryBuilder {
	q.where = where
	return q
}

func (q *alterMaskingPolicyQueryBuilder) GranteeNames(names []string) AlterMaskingPolicyQueryBuilder {
	q.granteeNames = names
	return q
}

func (q *alterMaskingPolicyQueryBuilder) GranteeAll(all bool) AlterMaskingPolicyQueryBuilder {
	q.granteeAll = all
	return q
}

func (q *alterMaskingPolicyQueryBuilder) GranteeAllExcept(except []string) AlterMaskingPolicyQueryBuilder {
	q.granteeAllExcept = except
	return q
}

func (q *alterMaskingPolicyQueryBuilder) WithPriority(priority *int64) AlterMaskingPolicyQueryBuilder {
	q.priority = priority
	return q
}

func (q *alterMaskingPolicyQueryBuilder) Build() (string, error) {
	if q.name == "" {
		return "", errors.New("masking policy name cannot be empty")
	}
	if q.database == "" || q.table == "" {
		return "", errors.New("database and table are required for masking policies")
	}

	grantees := granteeClause(q.granteeNames, q.granteeAll, q.granteeAllExcept)
	if grantees == "" {
		return "", errors.New("must specify at least one grantee: user, role, ALL, or ALL EXCEPT")
	}

	assignments, err := maskAssignments(q.masks)
	if err != nil {
		return "", err
	}

	tokens := []string{"ALTER", "MASKING", "POLICY", backtick(q.name), "ON", fmt.Sprintf("%s.%s", backtick(q.database), backtick(q.table))}

	if q.renameTo != nil && *q.renameTo != q.name {
		tokens = append(tokens, "RENAME", "TO", backtick(*q.renameTo))
	}

	tokens = append(tokens, "UPDATE", assignments)

	if q.where != nil && strings.TrimSpace(*q.where) != "" {
		tokens = append(tokens, "WHERE", *q.where)
	}

	tokens = append(tokens, "TO", grantees)

	if q.priority != nil {
		tokens = append(tokens, "PRIORITY", fmt.Sprintf("%d", *q.priority))
	}

	return strings.Join(tokens, " ") + ";", nil
}
