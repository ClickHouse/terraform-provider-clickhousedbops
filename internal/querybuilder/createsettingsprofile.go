package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// CreateSettingsProfileQueryBuilder is an interface to build CREATE SETTINGS PROFILE SQL queries (already interpolated).
type CreateSettingsProfileQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) CreateSettingsProfileQueryBuilder
}

type createSettingsProfileQueryBuilder struct {
	profileName string
	clusterName *string
}

func NewCreateSettingsProfile(name string) CreateSettingsProfileQueryBuilder {
	return &createSettingsProfileQueryBuilder{
		profileName: name,
	}
}

func (q *createSettingsProfileQueryBuilder) WithCluster(clusterName *string) CreateSettingsProfileQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *createSettingsProfileQueryBuilder) Build() (string, error) {
	if q.profileName == "" {
		return "", errors.New("profileName cannot be empty for CREATE SETTINGS PROFILE queries")
	}

	tokens := []string{
		"CREATE",
		"SETTINGS PROFILE",
		backtick(q.profileName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	return strings.Join(tokens, " ") + ";", nil
}
