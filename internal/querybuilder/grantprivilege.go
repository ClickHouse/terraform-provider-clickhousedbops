package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

// GrantPrivilegeQueryBuilder is an interface to build GRANT SQL queries (already interpolated).
type GrantPrivilegeQueryBuilder interface {
	QueryBuilder
	WithDatabase(*string) GrantPrivilegeQueryBuilder
	WithTable(*string) GrantPrivilegeQueryBuilder
	WithColumn(*string) GrantPrivilegeQueryBuilder
	WithGrantOption(bool) GrantPrivilegeQueryBuilder
	WithCluster(*string) GrantPrivilegeQueryBuilder
}

type grantPrivilegeQueryBuilder struct {
	accessType  string
	to          string
	database    *string
	table       *string
	column      *string
	grantOption bool
	clusterName *string
}

func GrantPrivilege(accessType string, to string) GrantPrivilegeQueryBuilder {
	return &grantPrivilegeQueryBuilder{
		accessType: accessType,
		to:         to,
	}
}

func (q *grantPrivilegeQueryBuilder) WithDatabase(database *string) GrantPrivilegeQueryBuilder {
	q.database = database
	return q
}

func (q *grantPrivilegeQueryBuilder) WithTable(table *string) GrantPrivilegeQueryBuilder {
	q.table = table
	return q
}

func (q *grantPrivilegeQueryBuilder) WithColumn(column *string) GrantPrivilegeQueryBuilder {
	q.column = column
	return q
}

func (q *grantPrivilegeQueryBuilder) WithCluster(clusterName *string) GrantPrivilegeQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *grantPrivilegeQueryBuilder) WithGrantOption(grantOption bool) GrantPrivilegeQueryBuilder {
	q.grantOption = grantOption
	return q
}

func (q *grantPrivilegeQueryBuilder) Build() (string, error) {
	if q.accessType == "" {
		return "", errors.New("AccessType cannot be empty")
	}
	if q.to == "" {
		return "", errors.New("To cannot be empty")
	}

	tokens := []string{
		"GRANT",
	}

	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	// Privilege
	if q.column != nil && *q.column != "" {
		tokens = append(tokens, fmt.Sprintf("%s(%s)", q.accessType, backtick(*q.column)))
	} else {
		tokens = append(tokens, q.accessType)
	}

	// Target database/table
	{
		tokens = append(tokens, "ON")

		if q.database != nil {
			if q.table != nil {
				tokens = append(tokens, fmt.Sprintf("%s.%s", backtick(*q.database), backtick(*q.table)))
			} else {
				tokens = append(tokens, fmt.Sprintf("%s.*", backtick(*q.database)))
			}
		} else {
			tokens = append(tokens, "*.*")
		}
	}

	// Grantee
	{
		tokens = append(tokens, "TO")
		tokens = append(tokens, backtick(q.to))
	}

	if q.grantOption {
		tokens = append(tokens, "WITH GRANT OPTION")
	}

	return strings.Join(tokens, " ") + ";", nil
}
