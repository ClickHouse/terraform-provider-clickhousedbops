package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// CreateRoleQueryBuilder is an interface to build CREATE ROLE SQL queries (already interpolated).
type CreateRoleQueryBuilder interface {
	QueryBuilder
	WithSettingsProfile(profileName *string) CreateRoleQueryBuilder
	WithCluster(clusterName *string) CreateRoleQueryBuilder
}

type createRoleQueryBuilder struct {
	resourceName    string
	settingsProfile *string
	clusterName     *string
}

func NewCreateRole(resourceName string) CreateRoleQueryBuilder {
	return &createRoleQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *createRoleQueryBuilder) WithSettingsProfile(profileName *string) CreateRoleQueryBuilder {
	q.settingsProfile = profileName
	return q
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
	if q.settingsProfile != nil {
		tokens = append(tokens, "SETTINGS", "PROFILE", quote(*q.settingsProfile))
	}

	return strings.Join(tokens, " ") + ";", nil
}
