package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// CreateSettingsProfileQueryBuilder is an interface to build CREATE SETTINGS PROFILE SQL queries (already interpolated).
type CreateSettingsProfileQueryBuilder interface {
	QueryBuilder
	WithInheritProfile(profileName *string) CreateSettingsProfileQueryBuilder
	WithCluster(clusterName *string) CreateSettingsProfileQueryBuilder
	AddSetting(name string, value *string, min *string, max *string, writability *string) CreateSettingsProfileQueryBuilder
}

type createSettingsProfileQueryBuilder struct {
	profileName    string
	clusterName    *string
	inheritProfile *string
	settings       []setting
}

func NewCreateSettingsProfile(name string) CreateSettingsProfileQueryBuilder {
	return &createSettingsProfileQueryBuilder{
		profileName: name,
		settings:    make([]setting, 0),
	}
}

func (q *createSettingsProfileQueryBuilder) WithCluster(clusterName *string) CreateSettingsProfileQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *createSettingsProfileQueryBuilder) WithInheritProfile(profileName *string) CreateSettingsProfileQueryBuilder {
	q.inheritProfile = profileName
	return q
}

func (q *createSettingsProfileQueryBuilder) AddSetting(name string, value *string, min *string, max *string, writability *string) CreateSettingsProfileQueryBuilder {
	q.settings = append(q.settings, &settingData{
		Name:        name,
		Value:       value,
		Min:         min,
		Max:         max,
		Writability: writability,
	})
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

	if len(q.settings) == 0 {
		return "", errors.New("cannot create settings profile with no settings")
	}

	tokens = append(tokens, "SETTINGS")

	renderedSettings := make([]string, 0)
	for _, s := range q.settings {
		def, err := s.SQLDef()
		if err != nil {
			return "", errors.WithMessage(err, "Error building query")
		}
		renderedSettings = append(renderedSettings, def)
	}

	tokens = append(tokens, strings.Join(renderedSettings, ", "))

	if q.inheritProfile != nil {
		tokens = append(tokens, "INHERIT", quote(*q.inheritProfile))
	}

	return strings.Join(tokens, " ") + ";", nil
}
