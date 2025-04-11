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

type createDropQueryBuilder struct {
	action           string
	resourceTypeName string
	resourceName     string
}

func NewCreateRole(resourceName string) QueryBuilder {
	return newCreate(resourceTypeRole, resourceName)
}

func NewDropRole(resourceName string) QueryBuilder {
	return newDrop(resourceTypeRole, resourceName)
}

func NewDropDatabase(resourceName string) QueryBuilder {
	return newDrop(resourceTypeDatabase, resourceName)
}

func NewDropUser(resourceName string) QueryBuilder {
	return newDrop(resourceTypeUser, resourceName)
}

func newCreate(resourceTypeName string, resourceName string) QueryBuilder {
	return &createDropQueryBuilder{
		action:           actionCreate,
		resourceTypeName: resourceTypeName,
		resourceName:     resourceName,
	}
}

func newDrop(resourceTypeName string, resourceName string) QueryBuilder {
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

	return strings.Join(tokens, " ") + ";", nil
}
