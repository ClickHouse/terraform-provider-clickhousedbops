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
	OrReplace(bool) CreateMaskingPolicyQueryBuilder
	IfNotExists(bool) CreateMaskingPolicyQueryBuilder
	WithWhere(*string) CreateMaskingPolicyQueryBuilder
	WithGrantees(users []string, roles []string, all bool, allExcept []string) CreateMaskingPolicyQueryBuilder
	WithPriority(*int64) CreateMaskingPolicyQueryBuilder
}

type createMaskingPolicyQueryBuilder struct {
	name        string
	database    string
	table       string
	masks       []ColumnMask
	where       *string
	users       []string
	roles       []string
	all         bool
	allExcept   []string
	priority    *int64
	orReplace   bool
	ifNotExists bool
}

func NewCreateMaskingPolicy(name string, database string, table string, masks []ColumnMask) CreateMaskingPolicyQueryBuilder {
	return &createMaskingPolicyQueryBuilder{
		name:     name,
		database: database,
		table:    table,
		masks:    masks,
	}
}

func (q *createMaskingPolicyQueryBuilder) OrReplace(v bool) CreateMaskingPolicyQueryBuilder {
	q.orReplace = v
	return q
}

func (q *createMaskingPolicyQueryBuilder) IfNotExists(v bool) CreateMaskingPolicyQueryBuilder {
	q.ifNotExists = v
	return q
}

func (q *createMaskingPolicyQueryBuilder) WithWhere(where *string) CreateMaskingPolicyQueryBuilder {
	q.where = where
	return q
}

func (q *createMaskingPolicyQueryBuilder) WithGrantees(users []string, roles []string, all bool, allExcept []string) CreateMaskingPolicyQueryBuilder {
	q.users = users
	q.roles = roles
	q.all = all
	q.allExcept = allExcept
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
	if q.orReplace && q.ifNotExists {
		return "", errors.New("cannot use both OR REPLACE and IF NOT EXISTS")
	}

	grantees, err := maskingPolicyGrantees(q.users, q.roles, q.all, q.allExcept)
	if err != nil {
		return "", err
	}

	// OR REPLACE goes between CREATE and the object type (CREATE OR REPLACE MASKING POLICY),
	// while IF NOT EXISTS goes after it (CREATE MASKING POLICY IF NOT EXISTS), matching ClickHouse.
	tokens := []string{"CREATE"}
	if q.orReplace {
		tokens = append(tokens, "OR", "REPLACE")
	}
	tokens = append(tokens, "MASKING", "POLICY")
	if q.ifNotExists {
		tokens = append(tokens, "IF", "NOT", "EXISTS")
	}
	tokens = append(tokens, backtick(q.name), "ON", fmt.Sprintf("%s.%s", backtick(q.database), backtick(q.table)))

	// Sort masks by column name so the generated SQL is deterministic regardless of map iteration order.
	masks := append([]ColumnMask(nil), q.masks...)
	sort.Slice(masks, func(i, j int) bool { return masks[i].Column < masks[j].Column })

	assignments := make([]string, 0, len(masks))
	for _, m := range masks {
		if m.Column == "" {
			return "", errors.New("column name in mask cannot be empty")
		}
		if strings.TrimSpace(m.Expression) == "" {
			return "", errors.Errorf("expression for column %q cannot be empty", m.Column)
		}
		assignments = append(assignments, fmt.Sprintf("%s = %s", backtick(m.Column), m.Expression))
	}
	tokens = append(tokens, "UPDATE", strings.Join(assignments, ", "))

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

// maskingPolicyGrantees builds the TO clause: a list of users/roles, or ALL, or ALL EXCEPT a list.
func maskingPolicyGrantees(users []string, roles []string, all bool, allExcept []string) (string, error) {
	if all {
		if len(allExcept) > 0 {
			return "ALL EXCEPT " + strings.Join(backtickAll(allExcept), ", "), nil
		}
		return "ALL", nil
	}

	grantees := make([]string, 0, len(users)+len(roles))
	grantees = append(grantees, backtickAll(users)...)
	grantees = append(grantees, backtickAll(roles)...)
	if len(grantees) == 0 {
		return "", errors.New("must specify at least one grantee: user, role, or ALL")
	}
	return strings.Join(grantees, ", "), nil
}
