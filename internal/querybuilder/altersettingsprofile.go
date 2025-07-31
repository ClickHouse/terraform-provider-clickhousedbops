package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// AlterSettingsProfileQueryBuilder is an interface to build ALTER SETTINGS PROFILE SQL queries (already interpolated).
type AlterSettingsProfileQueryBuilder interface {
	QueryBuilder
	RenameTo(newName *string) AlterSettingsProfileQueryBuilder
	AddSetting(name string, value *string, min *string, max *string, writability *string) AlterSettingsProfileQueryBuilder
	RemoveSetting(name string) AlterSettingsProfileQueryBuilder
	InheritFrom(profileNames []string) AlterSettingsProfileQueryBuilder
	WithCluster(clusterName *string) AlterSettingsProfileQueryBuilder
}

type alterSettingsProfileQueryBuilder struct {
	resourceName   string
	newName        *string
	settings       []settingData
	removeSettings []string
	clusterName    *string
	dropProfiles   bool
	inheritFrom    []string
}

func NewAlterSettingsProfile(resourceName string) AlterSettingsProfileQueryBuilder {
	return &alterSettingsProfileQueryBuilder{
		resourceName: resourceName,
		settings:     make([]settingData, 0),
	}
}

func (q *alterSettingsProfileQueryBuilder) RenameTo(newName *string) AlterSettingsProfileQueryBuilder {
	q.newName = newName

	return q
}

func (q *alterSettingsProfileQueryBuilder) AddSetting(name string, value *string, min *string, max *string, writability *string) AlterSettingsProfileQueryBuilder {
	q.settings = append(q.settings, settingData{
		Name:        name,
		Value:       value,
		Min:         min,
		Max:         max,
		Writability: writability,
	})

	return q
}

func (q *alterSettingsProfileQueryBuilder) RemoveSetting(name string) AlterSettingsProfileQueryBuilder {
	q.removeSettings = append(q.removeSettings, backtick(name))

	return q
}

func (q *alterSettingsProfileQueryBuilder) WithCluster(clusterName *string) AlterSettingsProfileQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *alterSettingsProfileQueryBuilder) InheritFrom(profileNames []string) AlterSettingsProfileQueryBuilder {
	q.dropProfiles = true
	q.inheritFrom = profileNames
	return q
}

func (q *alterSettingsProfileQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for ALTER ROLE queries")
	}

	tokens := []string{
		"ALTER",
		"SETTINGS",
		"PROFILE",
		backtick(q.resourceName),
	}

	if q.newName != nil && q.resourceName != *q.newName {
		tokens = append(tokens, "RENAME", "TO", backtick(*q.newName))
	}

	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	if q.dropProfiles {
		tokens = append(tokens, "DROP ALL PROFILES")
	}

	if len(q.removeSettings) > 0 {
		tokens = append(tokens, "DROP", "SETTINGS", strings.Join(q.removeSettings, ", "))
	}

	if len(q.settings) > 0 {
		tokens = append(tokens, "ADD", "SETTINGS")

		each := make([]string, 0)
		for _, s := range q.settings {
			sql, err := s.SQLDef()
			if err != nil {
				return "", errors.WithMessage(err, "invalid setting")
			}
			each = append(each, sql)
		}

		tokens = append(tokens, strings.Join(each, ", "))
	}

	if len(q.inheritFrom) > 0 {
		tokens = append(tokens, "INHERIT", strings.Join(backtickAll(q.inheritFrom), ", "))
	}

	return strings.Join(tokens, " ") + ";", nil
}
