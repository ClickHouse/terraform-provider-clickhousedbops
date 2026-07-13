package querybuilder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pingcap/errors"
)

// ColumnMask is a single `column = expression` assignment in a MASKING POLICY UPDATE clause.
// Expression is interpolated verbatim: it is a ClickHouse expression (e.g. a multiIf(...) call),
// not a literal value, so it must not be quoted.
type ColumnMask struct {
	Column     string
	Expression string
}

// CreateMaskingPolicyQueryBuilder builds CREATE MASKING POLICY statements. Masking policies are
// ClickHouse Cloud only and have no ON CLUSTER form.
type CreateMaskingPolicyQueryBuilder interface {
	QueryBuilder
	WithWhere(*string) CreateMaskingPolicyQueryBuilder
	GranteeNames([]string) CreateMaskingPolicyQueryBuilder
	GranteeAll(bool) CreateMaskingPolicyQueryBuilder
	GranteeAllExcept([]string) CreateMaskingPolicyQueryBuilder
	WithPriority(*int64) CreateMaskingPolicyQueryBuilder
}

type createMaskingPolicyQueryBuilder struct {
	name             string
	database         string
	table            string
	masks            []ColumnMask
	where            *string
	granteeNames     []string
	granteeAll       bool
	granteeAllExcept []string
	priority         *int64
}

func NewCreateMaskingPolicy(name string, database string, table string, masks []ColumnMask) CreateMaskingPolicyQueryBuilder {
	return &createMaskingPolicyQueryBuilder{
		name:     name,
		database: database,
		table:    table,
		masks:    masks,
	}
}

func (q *createMaskingPolicyQueryBuilder) WithWhere(where *string) CreateMaskingPolicyQueryBuilder {
	q.where = where
	return q
}

func (q *createMaskingPolicyQueryBuilder) GranteeNames(names []string) CreateMaskingPolicyQueryBuilder {
	q.granteeNames = names
	return q
}

func (q *createMaskingPolicyQueryBuilder) GranteeAll(all bool) CreateMaskingPolicyQueryBuilder {
	q.granteeAll = all
	return q
}

func (q *createMaskingPolicyQueryBuilder) GranteeAllExcept(except []string) CreateMaskingPolicyQueryBuilder {
	q.granteeAllExcept = except
	return q
}

func (q *createMaskingPolicyQueryBuilder) WithPriority(priority *int64) CreateMaskingPolicyQueryBuilder {
	q.priority = priority
	return q
}

func (q *createMaskingPolicyQueryBuilder) Build() (string, error) {
	if q.name == "" {
		return "", errors.New("masking policy name cannot be empty")
	}
	if q.database == "" || q.table == "" {
		return "", errors.New("database and table are required for masking policies")
	}
	if len(q.masks) == 0 {
		return "", errors.New("at least one column mask is required")
	}

	grantees := granteeClause(q.granteeNames, q.granteeAll, q.granteeAllExcept)
	if grantees == "" {
		return "", errors.New("must specify at least one grantee: user, role, ALL, or ALL EXCEPT")
	}

	assignments, err := maskAssignments(q.masks)
	if err != nil {
		return "", err
	}

	tokens := []string{
		"CREATE", "MASKING", "POLICY", backtick(q.name), "ON",
		fmt.Sprintf("%s.%s", backtick(q.database), backtick(q.table)), "UPDATE", assignments,
	}

	if q.where != nil && strings.TrimSpace(*q.where) != "" {
		tokens = append(tokens, "WHERE", *q.where)
	}

	tokens = append(tokens, "TO", grantees)

	if q.priority != nil {
		tokens = append(tokens, "PRIORITY", fmt.Sprintf("%d", *q.priority))
	}

	return strings.Join(tokens, " ") + ";", nil
}

// DropMaskingPolicyQueryBuilder builds DROP MASKING POLICY statements.
type DropMaskingPolicyQueryBuilder interface {
	QueryBuilder
	IfExists(bool) DropMaskingPolicyQueryBuilder
}

type dropMaskingPolicyQueryBuilder struct {
	name     string
	database string
	table    string
	ifExists bool
}

func NewDropMaskingPolicy(name string, database string, table string) DropMaskingPolicyQueryBuilder {
	return &dropMaskingPolicyQueryBuilder{
		name:     name,
		database: database,
		table:    table,
	}
}

func (q *dropMaskingPolicyQueryBuilder) IfExists(v bool) DropMaskingPolicyQueryBuilder {
	q.ifExists = v
	return q
}

func (q *dropMaskingPolicyQueryBuilder) Build() (string, error) {
	if q.name == "" {
		return "", errors.New("masking policy name cannot be empty")
	}
	if q.database == "" || q.table == "" {
		return "", errors.New("database and table are required for masking policies")
	}

	tokens := []string{"DROP", "MASKING", "POLICY"}
	if q.ifExists {
		tokens = append(tokens, "IF", "EXISTS")
	}
	tokens = append(tokens, backtick(q.name), "ON", fmt.Sprintf("%s.%s", backtick(q.database), backtick(q.table)))

	return strings.Join(tokens, " ") + ";", nil
}

// maskAssignments renders the UPDATE clause assignments, sorted by column name so the generated
// SQL is deterministic regardless of map iteration order.
func maskAssignments(masks []ColumnMask) (string, error) {
	if len(masks) == 0 {
		return "", errors.New("at least one column mask is required")
	}

	sorted := append([]ColumnMask(nil), masks...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Column < sorted[j].Column })

	assignments := make([]string, 0, len(sorted))
	for _, m := range sorted {
		if m.Column == "" {
			return "", errors.New("column name in mask cannot be empty")
		}
		if strings.TrimSpace(m.Expression) == "" {
			return "", errors.Errorf("expression for column %q cannot be empty", m.Column)
		}
		assignments = append(assignments, fmt.Sprintf("%s = %s", backtick(m.Column), m.Expression))
	}
	return strings.Join(assignments, ", "), nil
}
