package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

const (
	resourceTypeDatabase        = "DATABASE"
	resourceTypeRole            = "ROLE"
	resourceTypeUser            = "USER"
	resourceTypeSettingsProfile = "SETTINGS PROFILE"
)

type DropQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) DropQueryBuilder
}

type dropQueryBuilder struct {
	resourceTypeName string
	resourceName     string
	clusterName      *string
}

func NewDropRole(resourceName string) DropQueryBuilder {
	return newDrop(resourceTypeRole, resourceName)
}

func NewDropDatabase(resourceName string) DropQueryBuilder {
	return newDrop(resourceTypeDatabase, resourceName)
}

func NewDropUser(resourceName string) DropQueryBuilder {
	return newDrop(resourceTypeUser, resourceName)
}

func NewDropSettingsProfile(resourceName string) DropQueryBuilder {
	return newDrop(resourceTypeSettingsProfile, resourceName)
}

func (q *dropQueryBuilder) WithCluster(clusterName *string) DropQueryBuilder {
	q.clusterName = clusterName
	return q
}

func newDrop(resourceTypeName string, resourceName string) DropQueryBuilder {
	return &dropQueryBuilder{
		resourceTypeName: resourceTypeName,
		resourceName:     resourceName,
	}
}

func (q *dropQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for CREATE and DROP queries")
	}

	tokens := []string{
		"DROP",
		q.resourceTypeName,
		backtick(q.resourceName),
	}

	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	return strings.Join(tokens, " ") + ";", nil
}
