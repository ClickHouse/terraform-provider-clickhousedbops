package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// AlterUserQueryBuilder is an interface to build ALTER USER SQL queries (already interpolated).
type AlterUserQueryBuilder interface {
	QueryBuilder
	WithOldSettingProfile(profileName *string) AlterUserQueryBuilder
	WithNewSettingProfile(profileName *string) AlterUserQueryBuilder
	WithCluster(clusterName *string) AlterUserQueryBuilder
}

type alterUserQueryBuilder struct {
	resourceName       string
	oldSettingsProfile *string
	newSettingsProfile *string
	clusterName        *string
}

func NewAlterUser(resourceName string) AlterUserQueryBuilder {
	return &alterUserQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *alterUserQueryBuilder) WithOldSettingProfile(profileName *string) AlterUserQueryBuilder {
	q.oldSettingsProfile = profileName
	return q
}

func (q *alterUserQueryBuilder) WithNewSettingProfile(profileName *string) AlterUserQueryBuilder {
	q.newSettingsProfile = profileName
	return q
}

func (q *alterUserQueryBuilder) WithCluster(clusterName *string) AlterUserQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *alterUserQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for ALTER ROLE queries")
	}

	if (q.oldSettingsProfile == nil && q.newSettingsProfile == nil) ||
		(q.oldSettingsProfile != nil && q.newSettingsProfile != nil && *q.oldSettingsProfile == *q.newSettingsProfile) {
		return "", errors.New("no change to be made")
	}

	tokens := []string{
		"ALTER",
		"USER",
		backtick(q.resourceName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}
	if q.oldSettingsProfile != nil {
		tokens = append(tokens, "DROP", "PROFILES", quote(*q.oldSettingsProfile))
	}
	if q.newSettingsProfile != nil {
		tokens = append(tokens, "ADD", "PROFILES", quote(*q.newSettingsProfile))
	}

	return strings.Join(tokens, " ") + ";", nil
}
