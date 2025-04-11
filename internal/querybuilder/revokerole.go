package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// RevokeRoleQueryBuilder is an interface to build REVOKE SQL queries (already interpolated).
type RevokeRoleQueryBuilder interface {
	QueryBuilder
}

type revokeRoleQueryBuilder struct {
	roleName string
	from     string
}

func RevokeRole(roleName string, from string) RevokeRoleQueryBuilder {
	return &revokeRoleQueryBuilder{
		roleName: roleName,
		from:     from,
	}
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
		backtick(q.roleName),
		"FROM",
		backtick(q.from),
	}

	return strings.Join(tokens, " ") + ";", nil
}
