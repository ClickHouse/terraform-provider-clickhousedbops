package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// RevokeRoleQueryBuilder is an interface to build REVOKE SQL queries (already interpolated).
type RevokeRoleQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) RevokeRoleQueryBuilder
}

type revokeRoleQueryBuilder struct {
	roleName    string
	from        string
	clusterName *string
}

func RevokeRole(roleName string, from string) RevokeRoleQueryBuilder {
	return &revokeRoleQueryBuilder{
		roleName: roleName,
		from:     from,
	}
}

func (q *revokeRoleQueryBuilder) WithCluster(clusterName *string) RevokeRoleQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *revokeRoleQueryBuilder) Build() (string, error) {
	if q.roleName == "" {
		return "", errors.New("RoleName cannot be empty")
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

	tokens = append(tokens, backtick(q.roleName), "FROM", backtick(q.from))

	return strings.Join(tokens, " ") + ";", nil
}
