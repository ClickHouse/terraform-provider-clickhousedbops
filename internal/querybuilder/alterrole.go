package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// AlterRoleQueryBuilder is an interface to build ALTER ROLE SQL queries (already interpolated).
type AlterRoleQueryBuilder interface {
	QueryBuilder
	WithOldSettingProfile(profileName *string) AlterRoleQueryBuilder
	WithNewSettingProfile(profileName *string) AlterRoleQueryBuilder
	WithCluster(clusterName *string) AlterRoleQueryBuilder
}

type alterRoleQueryBuilder struct {
	resourceName       string
	oldSettingsProfile *string
	newSettingsProfile *string
	clusterName        *string
}

func NewAlterRole(resourceName string) AlterRoleQueryBuilder {
	return &alterRoleQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *alterRoleQueryBuilder) WithOldSettingProfile(profileName *string) AlterRoleQueryBuilder {
	q.oldSettingsProfile = profileName
	return q
}

func (q *alterRoleQueryBuilder) WithNewSettingProfile(profileName *string) AlterRoleQueryBuilder {
	q.newSettingsProfile = profileName
	return q
}

func (q *alterRoleQueryBuilder) WithCluster(clusterName *string) AlterRoleQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *alterRoleQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for ALTER ROLE queries")
	}

	if (q.oldSettingsProfile == nil && q.newSettingsProfile == nil) ||
		(q.oldSettingsProfile != nil && q.newSettingsProfile != nil && *q.oldSettingsProfile == *q.newSettingsProfile) {
		return "", errors.New("no change to be made")
	}

	tokens := []string{
		"ALTER",
		"ROLE",
		backtick(q.resourceName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}
	if q.oldSettingsProfile != nil {
		tokens = append(tokens, "DROP", "PROFILES", quote(*q.oldSettingsProfile))
	}
	if q.newSettingsProfile != nil {
		tokens = append(tokens, "ADD", "PROFILE", quote(*q.newSettingsProfile))
	}

	return strings.Join(tokens, " ") + ";", nil
}
