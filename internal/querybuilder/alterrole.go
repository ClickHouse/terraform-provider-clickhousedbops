package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// AlterRoleQueryBuilder is an interface to build ALTER ROLE SQL queries (already interpolated).
type AlterRoleQueryBuilder interface {
	QueryBuilder
	RenameTo(newName *string) AlterRoleQueryBuilder
	DropSettingsProfile(profileName *string) AlterRoleQueryBuilder
	AddSettingsProfile(profileName *string) AlterRoleQueryBuilder
	WithCluster(clusterName *string) AlterRoleQueryBuilder
}

type alterRoleQueryBuilder struct {
	resourceName       string
	oldSettingsProfile *string
	newSettingsProfile *string
	newName            *string
	clusterName        *string
}

func NewAlterRole(resourceName string) AlterRoleQueryBuilder {
	return &alterRoleQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *alterRoleQueryBuilder) RenameTo(newName *string) AlterRoleQueryBuilder {
	q.newName = newName

	return q
}

func (q *alterRoleQueryBuilder) DropSettingsProfile(profileName *string) AlterRoleQueryBuilder {
	q.oldSettingsProfile = profileName
	return q
}

func (q *alterRoleQueryBuilder) AddSettingsProfile(profileName *string) AlterRoleQueryBuilder {
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

	anyChanges := false

	tokens := []string{
		"ALTER",
		"ROLE",
		backtick(q.resourceName),
	}
	if q.newName != nil && *q.newName != q.resourceName {
		anyChanges = true
		tokens = append(tokens, "RENAME", "TO", backtick(*q.newName))
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}
	if (q.oldSettingsProfile != nil && q.newSettingsProfile != nil && *q.oldSettingsProfile != *q.newSettingsProfile) ||
		(q.oldSettingsProfile == nil && q.newSettingsProfile != nil) ||
		(q.oldSettingsProfile != nil && q.newSettingsProfile == nil) {
		// Settings profile was changed
		anyChanges = true
		if q.oldSettingsProfile != nil {
			tokens = append(tokens, "DROP", "PROFILES", quote(*q.oldSettingsProfile))
		}
		if q.newSettingsProfile != nil {
			tokens = append(tokens, "ADD", "PROFILE", quote(*q.newSettingsProfile))
		}
	}

	if !anyChanges {
		return "", errors.New("no change to be made")
	}

	return strings.Join(tokens, " ") + ";", nil
}
