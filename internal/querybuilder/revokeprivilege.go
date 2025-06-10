package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

// RevokePrivilegeQueryBuilder is an interface to build REVOKE SQL queries (already interpolated).
type RevokePrivilegeQueryBuilder interface {
	QueryBuilder
	WithDatabase(*string) RevokePrivilegeQueryBuilder
	WithTable(*string) RevokePrivilegeQueryBuilder
	WithColumn(*string) RevokePrivilegeQueryBuilder
	WithCluster(*string) RevokePrivilegeQueryBuilder
}

type revokePrivilegeQueryBuilder struct {
	accessType  string
	from        string
	database    *string
	table       *string
	column      *string
	clusterName *string
}

func RevokePrivilege(accessType string, from string) RevokePrivilegeQueryBuilder {
	return &revokePrivilegeQueryBuilder{
		accessType: accessType,
		from:       from,
	}
}

func (q *revokePrivilegeQueryBuilder) WithDatabase(database *string) RevokePrivilegeQueryBuilder {
	q.database = database
	return q
}

func (q *revokePrivilegeQueryBuilder) WithTable(table *string) RevokePrivilegeQueryBuilder {
	q.table = table
	return q
}

func (q *revokePrivilegeQueryBuilder) WithColumn(column *string) RevokePrivilegeQueryBuilder {
	q.column = column
	return q
}

func (q *revokePrivilegeQueryBuilder) WithCluster(clusterName *string) RevokePrivilegeQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *revokePrivilegeQueryBuilder) Build() (string, error) {
	if q.accessType == "" {
		return "", errors.New("AccessType cannot be empty")
	}
	if q.from == "" {
		return "", errors.New("From cannot be empty")
	}

	tokens := []string{
		"REVOKE",
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
		tokens = append(tokens, "FROM")
		tokens = append(tokens, backtick(q.from))
	}

	return strings.Join(tokens, " ") + ";", nil
}
