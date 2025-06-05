package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

const (
	resourceTypeDatabase = "DATABASE"
	resourceTypeRole     = "ROLE"
	resourceTypeUser     = "USER"

	actionCreate = "CREATE"
	actionDrop   = "DROP"
)

type CreateDropQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) CreateDropQueryBuilder
}

type createDropQueryBuilder struct {
	action           string
	resourceTypeName string
	resourceName     string
	clusterName      *string
}

func NewCreateRole(resourceName string) CreateDropQueryBuilder {
	return newCreate(resourceTypeRole, resourceName)
}

func NewDropRole(resourceName string) CreateDropQueryBuilder {
	return newDrop(resourceTypeRole, resourceName)
}

func NewDropDatabase(resourceName string) CreateDropQueryBuilder {
	return newDrop(resourceTypeDatabase, resourceName)
}

func NewDropUser(resourceName string) CreateDropQueryBuilder {
	return newDrop(resourceTypeUser, resourceName)
}

func (q *createDropQueryBuilder) WithCluster(clusterName *string) CreateDropQueryBuilder {
	q.clusterName = clusterName
	return q
}

func newCreate(resourceTypeName string, resourceName string) CreateDropQueryBuilder {
	return &createDropQueryBuilder{
		action:           actionCreate,
		resourceTypeName: resourceTypeName,
		resourceName:     resourceName,
	}
}

func newDrop(resourceTypeName string, resourceName string) CreateDropQueryBuilder {
	return &createDropQueryBuilder{
		action:           actionDrop,
		resourceTypeName: resourceTypeName,
		resourceName:     resourceName,
	}
}

func (q *createDropQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for CREATE and DROP queries")
	}

	tokens := []string{
		q.action,
		q.resourceTypeName,
		backtick(q.resourceName),
	}

	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	return strings.Join(tokens, " ") + ";", nil
}
