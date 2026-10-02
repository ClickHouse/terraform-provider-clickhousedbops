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
	WithAccessObject(*string) GrantPrivilegeQueryBuilder
	WithAccessObjectFilter(*string) GrantPrivilegeQueryBuilder
	WithParameterizedTarget(bool) GrantPrivilegeQueryBuilder
	WithGrantOption(bool) GrantPrivilegeQueryBuilder
	WithCluster(*string) GrantPrivilegeQueryBuilder
	WithCurrentGrants(bool) GrantPrivilegeQueryBuilder
}

type grantPrivilegeQueryBuilder struct {
	accessType          string
	to                  string
	database            *string
	table               *string
	column              *string
	accessObject        *string
	accessObjectFilter  *string
	parameterizedTarget bool
	grantOption         bool
	clusterName         *string
	currentGrants       bool
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

func (q *grantPrivilegeQueryBuilder) WithAccessObject(accessObject *string) GrantPrivilegeQueryBuilder {
	q.accessObject = accessObject
	return q
}

func (q *grantPrivilegeQueryBuilder) WithAccessObjectFilter(filter *string) GrantPrivilegeQueryBuilder {
	q.accessObjectFilter = filter
	return q
}

func (q *grantPrivilegeQueryBuilder) WithParameterizedTarget(parameterized bool) GrantPrivilegeQueryBuilder {
	q.parameterizedTarget = parameterized
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

func (q *grantPrivilegeQueryBuilder) WithCurrentGrants(currentGrants bool) GrantPrivilegeQueryBuilder {
	q.currentGrants = currentGrants
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
	var privilege string
	if q.column != nil && *q.column != "" {
		privilege = fmt.Sprintf("%s(%s)", q.accessType, backtick(*q.column))
	} else {
		privilege = q.accessType
	}

	target, err := privilegeTarget(q.database, q.table, q.accessObject, q.accessObjectFilter, q.parameterizedTarget)
	if err != nil {
		return "", err
	}

	// CURRENT GRANTS copies the grantor's own privileges. ClickHouse Cloud requires it for
	// broad grants (e.g. ALL, SELECT ON *.*) the default admin holds but cannot transfer
	// directly. See ClickHouse/terraform-provider-clickhousedbops#190.
	if q.currentGrants {
		tokens = append(tokens, fmt.Sprintf("CURRENT GRANTS(%s ON %s)", privilege, target))
	} else {
		tokens = append(tokens, privilege, "ON", target)
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
