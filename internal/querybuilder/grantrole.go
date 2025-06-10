package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// GrantRoleQueryBuilder is an interface to build GRANT SQL queries (already interpolated).
type GrantRoleQueryBuilder interface {
	QueryBuilder
	WithAdminOption(bool) GrantRoleQueryBuilder
	WithCluster(clusterName *string) GrantRoleQueryBuilder
}

type grantQueryBuilder struct {
	roleName    string
	to          string
	adminOption bool
	clusterName *string
}

func GrantRole(roleName string, to string) GrantRoleQueryBuilder {
	return &grantQueryBuilder{
		roleName: roleName,
		to:       to,
	}
}

func (q *grantQueryBuilder) WithAdminOption(adminOption bool) GrantRoleQueryBuilder {
	q.adminOption = adminOption
	return q
}

func (q *grantQueryBuilder) WithCluster(clusterName *string) GrantRoleQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *grantQueryBuilder) Build() (string, error) {
	if q.roleName == "" {
		return "", errors.New("RoleName cannot be empty")
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

	tokens = append(tokens, backtick(q.roleName), "TO", backtick(q.to))

	if q.adminOption {
		tokens = append(tokens, "WITH ADMIN OPTION")
	}

	return strings.Join(tokens, " ") + ";", nil
}
