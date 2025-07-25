package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// CreateRoleQueryBuilder is an interface to build CREATE ROLE SQL queries (already interpolated).
type CreateRoleQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) CreateRoleQueryBuilder
}

type createRoleQueryBuilder struct {
	resourceName string
	clusterName  *string
}

func NewCreateRole(resourceName string) CreateRoleQueryBuilder {
	return &createRoleQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *createRoleQueryBuilder) WithCluster(clusterName *string) CreateRoleQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *createRoleQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for CREATE ROLE queries")
	}

	tokens := []string{
		"CREATE",
		"ROLE",
		backtick(q.resourceName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	return strings.Join(tokens, " ") + ";", nil
}
