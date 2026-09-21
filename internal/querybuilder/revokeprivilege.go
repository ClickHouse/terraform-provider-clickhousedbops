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
	WithAccessObject(*string) RevokePrivilegeQueryBuilder
	WithCluster(*string) RevokePrivilegeQueryBuilder
	WithGrantOptionOnly(bool) RevokePrivilegeQueryBuilder
}

type revokePrivilegeQueryBuilder struct {
	accessType      string
	from            string
	database        *string
	table           *string
	column          *string
	accessObject    *string
	clusterName     *string
	grantOptionOnly bool
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

func (q *revokePrivilegeQueryBuilder) WithAccessObject(accessObject *string) RevokePrivilegeQueryBuilder {
	q.accessObject = accessObject
	return q
}

func (q *revokePrivilegeQueryBuilder) WithCluster(clusterName *string) RevokePrivilegeQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *revokePrivilegeQueryBuilder) WithGrantOptionOnly(grantOptionOnly bool) RevokePrivilegeQueryBuilder {
	q.grantOptionOnly = grantOptionOnly
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

	if q.grantOptionOnly {
		tokens = append(tokens, "GRANT OPTION FOR")
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

		switch {
		case q.accessObject != nil:
			tokens = append(tokens, identifierOrPattern(*q.accessObject))
		case q.database != nil && q.table != nil:
			tokens = append(tokens, fmt.Sprintf("%s.%s", identifierOrPattern(*q.database), identifierOrPattern(*q.table)))
		case q.database != nil:
			tokens = append(tokens, fmt.Sprintf("%s.*", identifierOrPattern(*q.database)))
		default:
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
